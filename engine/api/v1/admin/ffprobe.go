package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func GetFFprobe(ctx *gin.Context, db *gorm.DB, app *gin.Engine) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	fileId := ctx.Param("file_id")
	fileIdint, err := strconv.Atoi(fileId)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid file_id"})
		return
	}
	var file engine.FILE
	if tx := db.Where("id = ?", fileIdint).Preload("TORRENT").First(&file); tx.Error != nil {
		ctx.JSON(404, gin.H{"error": "file not found"})
		return
	}
	ffprobe, err := file.FfprobeData(app)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "ffprobe failed"})
		return
	}
	ctx.JSON(200, ffprobe)
}
