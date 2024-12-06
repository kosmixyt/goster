package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func Rescan(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(401, gin.H{"error": "not admin"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			ctx.JSON(500, gin.H{"error": "scan failed" + r.(string)})
			return
		}

		ctx.JSON(
			200,
			gin.H{"status": "ok"},
		)
	}()

	engine.Scan(engine.Config.Locations, db)
}
