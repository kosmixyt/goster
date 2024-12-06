package share

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
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
