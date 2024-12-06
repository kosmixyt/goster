package watching

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func DeleteFromWatchingList(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(400, gin.H{"error": "user not found"})
		return
	}
	elementType := ctx.Query("type")
	if elementType != engine.Tv && elementType != engine.Movie {
		ctx.JSON(400, gin.H{"error": "type is required"})
		return
	}
	provider, id, err := engine.ParseIdProvider(ctx.Query("uuid"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if provider != "db" {
		ctx.JSON(400, gin.H{"error": "invalid provider"})
		return
	}
	field := elementType + "_id"
	var watching engine.WATCHING
	if tx := db.Where("user_id = ? AND "+field+" = ?", user.ID, id).First(&watching); tx.Error != nil {
		ctx.JSON(400, gin.H{"error": "element not found"})
		return
	}
	db.Delete(&watching)
	ctx.JSON(200, gin.H{"status": "ok"})
}
