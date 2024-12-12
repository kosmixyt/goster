package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AdminActionController(user engine.User, action string) error {

	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	switch action {
	case "ffprobe":
	default:
		return engine.ErrorInvalidAction
	}
	return nil
}

func AdminAction(ctx *gin.Context, db *gorm.DB, app *gin.Engine) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err := AdminActionController(user, ctx.Param("action")); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}
