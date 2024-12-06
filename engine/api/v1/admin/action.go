package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AdminAction(ctx *gin.Context, db *gorm.DB, app *gin.Engine) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	action := ctx.Param("action")
	switch action {
	case "ffprobe":
	default:
		ctx.JSON(404, gin.H{"error": "action not found"})
		return

	}
}
