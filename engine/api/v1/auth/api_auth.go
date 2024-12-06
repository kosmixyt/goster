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
