package share

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func AddShare(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	share, err := AddShareController(db, &user, ctx.Query("id"))
	if err != nil {
		ctx.JSON(400, gin.H{
			"status": "error",
			"error":  err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "success", "share": gin.H{"id": share.ID, "expire": share.EXPIRE}})

}

func AddShareController(db *gorm.DB, user *engine.User, file_id string) (*engine.Share, error) {
	var file *engine.FILE
	if err := db.Where("id = ?", file_id).First(&file).Error; err != nil {
		return nil, err
	}

	expirer := time.Duration(int64(24)) * time.Hour
	return user.NewShare(&expirer, *file), nil
}

func AddShareWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKeys([]string{"id"}, request.Options)
	share, err := AddShareController(db, &user, key["id"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"status": "success", "share": gin.H{"id": share.ID, "expire": share.EXPIRE}}, nil, request.RequestUuid)
}
