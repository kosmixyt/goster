package dlrequest

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func DeleteRequest(db *gorm.DB, ctx *gin.Context) {
	user, err := engine.GetUser(db, ctx, []string{"Requests"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return

	}
	if err = DeleteRequestController(&user, ctx.Query("id"), db); err == nil {
		ctx.JSON(200, gin.H{"status": "success"})
		return
	}
	ctx.JSON(400, gin.H{"error": "request not found"})
}
func DeleteRequestController(user *engine.User, str_id string, db *gorm.DB) error {
	id, err := strconv.ParseUint(str_id, 10, 64)
	if err != nil {
		return err
	}
	for _, req := range user.Requests {
		if uint64(req.ID) == id {
			db.Where("id = ?", req.ID).Delete(&engine.DownloadRequest{})
			return nil
		}
	}
	return errors.New("request not found")
}
func DeleteRequestWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{"Requests"})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKeys([]string{"id"}, request.Options)
	if err = DeleteRequestController(&user, key["id"], db); err == nil {
		kosmixutil.SendWebsocketResponse(conn, gin.H{"status": "success"}, nil, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, nil, errors.New("request not found"), request.RequestUuid)
}
