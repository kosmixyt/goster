package torrents

import (
	"archive/zip"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TorrentZipDownload(ctx *gin.Context, db *gorm.DB) {
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
	fmt.Println(uint(torrentId), user.ID)
	torrent := user.GetUserTorrent(uint(torrentId))
	if torrent == nil {
		ctx.JSON(404, gin.H{"error": "torrent not found"})
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename="+torrent.Torrent.Name()+".zip")
	ctx.Header("Content-Type", "application/zip")
	zipWriter := zip.NewWriter(ctx.Writer)
	defer zipWriter.Close()
	for _, file := range torrent.Torrent.Files() {
		zipFile, err := zipWriter.Create(file.DisplayPath())
		if err != nil {
			panic(err)
		}
		fileReader := file.NewReader()
		io.Copy(zipFile, fileReader)
		fmt.Println("Added file to zip", file.DisplayPath())
	}
}
