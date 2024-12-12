package iptv

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func Logo(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	b, err := LogoController(&user, ctx.Query("id"), ctx.Query("channel"), db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.Data(200, "image/png", b)
	b = nil
}
func LogoController(user *engine.User, iptv_id string, channel_id string, db *gorm.DB) ([]byte, error) {
	iptvId, err := strconv.Atoi(iptv_id)
	if err != nil {
		return nil, errors.New("invalid id")
	}
	iptv := user.GetIptvById(iptvId)
	if iptv == nil {
		return nil, errors.New("iptv not found")
	}
	channelId, err := strconv.Atoi(channel_id)
	if err != nil {
		return nil, errors.New("invalid channel")
	}
	channel := iptv.GetChannel(channelId)
	if channel == nil {
		return nil, errors.New("channel not found")
	}
	url := channel.Logo_url
	name := base64.StdEncoding.EncodeToString([]byte(channel.Name))
	finalFilename := engine.Joins(engine.IMG_PATH, name+".png")
	if _, err := os.Stat(finalFilename); errors.Is(err, os.ErrNotExist) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, errors.New("server error")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.New("server error")
		}
		f, err := os.Create(finalFilename)
		if err != nil {
			return nil, errors.New("server error")
		}
		defer f.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New("server error")
		}
		_, err = f.Write(data)
		if err != nil {
			return nil, errors.New("server error")
		}
		return data, nil
	}
	file, err := os.Open(finalFilename)
	if err != nil {
		return nil, errors.New("server error")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.New("server error")
	}
	return data, nil
}
func LogoWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	vals := kosmixutil.GetStringKeys([]string{"id", "channel"}, request.Options)
	if data, err := LogoController(&user, vals["id"], vals["channel"], db); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
	} else {
		kosmixutil.SendWebsocketResponse(conn, data, nil, request.RequestUuid)
	}
}
