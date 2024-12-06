package admin

import (
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func LoginAsUser(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	userIdStr := ctx.Query("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	var target_user engine.User
	if tx := db.Where("id = ?", userId).First(&target_user); tx.Error != nil {
		ctx.JSON(404, gin.H{"error": "user not found"})
		return
	}
	session := sessions.Default(ctx)
	session.Set("user_id", target_user.ID)
	session.Save()
	ctx.JSON(200, gin.H{"status": "ok"})
}
