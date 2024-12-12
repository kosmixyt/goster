package metadata

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func BulkSerieMoveController(user *engine.User, db *gorm.DB, source_id_str string, target_id_str string) error {
	if !user.CAN_EDIT {
		return fmt.Errorf("forbidden")
	}
	source_id := source_id_str
	target_id := target_id_str
	fmt.Println("source_id", source_id, "target_id", target_id)
	source, err := engine.Get_tv_via_provider(source_id, true, user.RenderTvPreloads)
	if err != nil {
		return err
	}
	target, err := engine.Get_tv_via_provider(target_id, true, user.RenderTvPreloads)
	if err != nil {
		return err
	}
	if source.ID == target.ID {
		return fmt.Errorf("source and target are the same")
	}
	if err := source.MoveFiles(target); err != nil {
		return err
	}
	return nil
}
func BulkSerieMove(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	err = BulkSerieMoveController(&user, db, ctx.PostForm("source_id"), ctx.PostForm("target_id"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}
func BulkSerieMoveWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"source_id", "target_id"}, request.Options)
	err = BulkSerieMoveController(&user, db, keys["source_id"], keys["target_id"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"message": "ok"}, nil, request.RequestUuid)
}
