package torrents

import (
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TorrentFile(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	torrentId, err := strconv.Atoi(ctx.Query("id"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	torrent := user.GetUserTorrent(uint(torrentId))
	if torrent == nil {
		ctx.JSON(404, gin.H{"error": "torrent not found"})
		return
	}
	var torrentDbItem *engine.Torrent
	if err := db.Where("id = ?", torrent.DB_ID).First(&torrentDbItem).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "could not find torrent in database"})
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename="+torrent.Torrent.Name()+".torrent")
	// ctx.File(torrentDbItem.PATH)
	ctx.File(filepath.Join(engine.FILES_TORRENT_PATH, torrentDbItem.PATH))
}
