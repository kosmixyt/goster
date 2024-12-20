package auth

import (
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

var banned_ips = map[string]int{}

func Login(ctx *gin.Context, db *gorm.DB) {
	session := sessions.Default(ctx)
	uuid := ctx.Query("uuid")
	ip_client := ctx.ClientIP()
	if session.Get("user_id") != nil {
		fmt.Println(session.Get("user_id"))
		ctx.JSON(200, gin.H{"status": "ok already logged in"})
		return
	}
	if uuid == "" {
		ctx.JSON(400, gin.H{"error": "no uuid"})
		return
	}
	var user engine.User
	if err := db.Where("token = ?", uuid).First(&user).Error; err != nil {
		if banned_ips[ip_client] > 5 {
			ctx.JSON(500, gin.H{"error": "too many requests"})
			return
		}

		banned_ips[ip_client]++
		ctx.JSON(400, gin.H{"error": "user not found"})
		return
	}
	session.Set("user_id", user.ID)
	session.Save()
	ctx.JSON(200, gin.H{"status": "ok"})
}
func Logout(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Clear()
	session.Options(sessions.Options{MaxAge: -1})
	session.Save()
	ctx.JSON(200, gin.H{"status": "ok"})
}
func UpdateToken(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	previousToken := ctx.PostForm("previousToken")
	newToken := ctx.PostForm("newToken")
	if previousToken == "" || newToken == "" {
		ctx.JSON(400, gin.H{"error": "missing parameters"})
		return
	}
	if err := db.Model(&user).Where("id = ? AND token = ?", user.ID, previousToken).Update("token", newToken); err.Error != nil {
		ctx.JSON(500, gin.H{"error": err.Error.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "If its good token is updated"})
}
