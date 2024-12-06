package transcode

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func Stop(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	var t *engine.Transcoder
	for _, tr := range engine.Transcoders {
		if tr.UUID == uuid && user.ID == tr.OWNER_ID {
			t = tr
			break
		}
	}
	if t == nil {
		ctx.JSON(404, gin.H{"error": "transcoder not found"})
		return
	}
	t.Destroy("Stopped By FrontEnd")

	ctx.JSON(200, gin.H{"status": "ok"})
}
