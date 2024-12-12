package watching

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func DeleteFromWatchingListController(user *engine.User, db *gorm.DB, elementType string, uuid string) error {
	if elementType != engine.Tv && elementType != engine.Movie {
		return fmt.Errorf("invalid type")
	}
	provider, id, err := engine.ParseIdProvider(uuid)
	if err != nil {
		return err
	}
	if provider != "db" {
		return fmt.Errorf("invalid provider")
	}
	field := elementType + "_id"
	if tx := db.Where("user_id = ? AND "+field+" = ?", user.ID, id).Delete(&engine.WATCHING{}); tx.Error != nil {
		return tx.Error
	}
	return nil
}

func DeleteFromWatchingList(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err := DeleteFromWatchingListController(&user, db, ctx.Query("type"), ctx.Query("id")); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"success": true})
}
func DeleteFromWatchingListWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKey("id", request.Options)
	if err := DeleteFromWatchingListController(&user, db, request.Type, key); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"success": true}, nil, request.RequestUuid)
}
