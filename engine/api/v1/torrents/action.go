package torrents

import (
	"strconv"

	"github.com/anacrolix/torrent/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TorrentAction(ctx *gin.Context, db *gorm.DB) {
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
	action := ctx.Query("action")
	switch action {
	case "pause":
		if torrentDbItem.Paused {
			ctx.JSON(200, gin.H{"status": "already paused"})
			return
		}
		torrent.Torrent.DisallowDataDownload()
		torrent.Torrent.DisallowDataUpload()
		torrentDbItem.Paused = true
		if tx := db.Save(&torrentDbItem); tx.Error != nil {
			ctx.JSON(500, gin.H{"status": "could not pause torrent"})
			return
		}
		ctx.JSON(200, gin.H{"status": "paused"})
	case "resume":
		if !torrentDbItem.Paused {
			ctx.JSON(200, gin.H{"status": "already resumed"})
			return
		}
		torrent.Torrent.AllowDataDownload()
		torrent.Torrent.AllowDataUpload()
		torrentDbItem.Paused = false
		if tx := db.Save(&torrentDbItem); tx.Error != nil {
			ctx.JSON(500, gin.H{"status": "could not resume torrent"})
			return
		}
		ctx.JSON(200, gin.H{"status": "resumed"})
	case "delete":
		if !user.ADMIN {
			ctx.JSON(403, gin.H{"error": "not allowed"})
			return
		}
		if err := engine.CleanDeleteTorrent(ctx.Query("deleteFiles") == "true", torrent, db); err != nil {
			ctx.JSON(500, gin.H{"error": "could not delete torrent" + err.Error()})
			return
		}
		ctx.JSON(200, gin.H{"success": "torrent deleted"})
	case "recheck":
		if !user.ADMIN {
			ctx.JSON(403, gin.H{"error": "not allowed"})
			return
		}
		go torrent.Torrent.VerifyData()
		ctx.JSON(200, gin.H{"status": "rechecking"})
	case "download":
		if !user.ADMIN {
			ctx.JSON(403, gin.H{"error": "not allowed"})
			return
		}
		for _, file := range torrent.Torrent.Files() {
			file.Download()
		}
		torrent.Torrent.DownloadAll()
		ctx.JSON(200, gin.H{"status": "downloading"})
	case "priority":
		if !user.ADMIN {
			ctx.JSON(403, gin.H{"error": "not allowed"})
			return
		}
		fileIndex, err := strconv.Atoi(ctx.Query("fileIndex"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid fileIndex"})
			return
		}
		priority, err := strconv.Atoi(ctx.Query("priority"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid priority"})
			return
		}
		if fileIndex < 0 || fileIndex >= len(torrent.Torrent.Files()) {
			ctx.JSON(400, gin.H{"error": "invalid fileIndex"})
			return
		}
		if priority < 0 || priority > 3 {
			ctx.JSON(400, gin.H{"error": "invalid priority"})
			return
		}
		file := torrent.Torrent.Files()[fileIndex]
		switch priority {
		case 0:
			file.SetPriority(types.PiecePriorityNone)
		case 1:
			file.SetPriority(types.PiecePriorityNormal)
		case 2:
			file.SetPriority(types.PiecePriorityHigh)
		case 3:
			file.SetPriority(types.PiecePriorityNow)
		default:
			ctx.JSON(400, gin.H{"error": "invalid priority"})
			return
		}
		ctx.JSON(200, gin.H{"status": "priority set"})

	default:
		ctx.JSON(400, gin.H{"error": "invalid action"})
	}
}
