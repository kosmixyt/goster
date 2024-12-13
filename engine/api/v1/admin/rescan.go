package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func RescanController(user *engine.User, channel chan error, db *gorm.DB) {
	if !user.ADMIN {
		channel <- engine.ErrorIsNotAdmin
		return
	}
	defer func() {
		if r := recover(); r != nil {
			channel <- engine.ErrorRescanFailed
			return
		}
	}()

	engine.Scan(db)
}

func Rescan(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	channel := make(chan error)
	go RescanController(&user, channel, db)
	err = <-channel
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}
