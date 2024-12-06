package torrents

import (
	"path"
	"strconv"

	"github.com/anacrolix/torrent/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func TorrentFileDownload(ctx *gin.Context, db *gorm.DB) {
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
	index, err := strconv.Atoi(ctx.Query("index"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid index"})
		return
	}
	if index >= len(torrent.Torrent.Files()) || index < 0 {
		ctx.JSON(400, gin.H{"error": "invalid index"})
		return
	}
	file := torrent.Torrent.Files()[index]
	file.Torrent().AllowDataDownload()

	file.SetPriority(types.PiecePriorityNow)
	fileName := path.Base(file.DisplayPath())
	ctx.Header("Content-Disposition", "attachment; filename="+fileName)
	kosmixutil.ServerRangeRequest(ctx, file.Length(), file.NewReader(), true, true)

}
