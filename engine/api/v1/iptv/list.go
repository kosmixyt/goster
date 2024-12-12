package iptv

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func ListIptv(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	channels, err := ListIptvController(&user, ctx.Query("offset"), ctx.Query("limit"), ctx.Query("id"), ctx.Query("group"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid parameters"})
		return
	}
	ctx.JSON(200, gin.H{"channels": engine.MapIptvToRender(channels)})

}
func ListIptvWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	vals := kosmixutil.GetStringKeys([]string{"offset", "limit", "id", "group"}, request.Options)
	channels, err := ListIptvController(&user, vals["offset"], vals["limit"], vals["id"], vals["group"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("invalid parameters"), request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, engine.MapIptvToRender(channels), nil, request.RequestUuid)
}
func ListIptvController(user *engine.User, offset string, limit string, id string, group string) ([]*engine.IptvChannel, error) {
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		offsetInt = 0
	}
	if offsetInt < 0 {
		offsetInt = 0
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = 20
	}
	if limitInt > 100 {
		limitInt = 100
	}
	iptvId, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	iptv := user.GetIptvById(iptvId)
	if iptv == nil {
		return nil, err
	}
	return iptv.ListIptv(int64(offsetInt), int64(limitInt), &group), nil
}
