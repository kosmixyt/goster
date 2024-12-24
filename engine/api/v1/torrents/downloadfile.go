package torrents

import (
	"errors"
	"io"
	"path"
	"strconv"

	"github.com/anacrolix/torrent/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

type FileReturn struct {
	FileName string
	Reader   io.ReadSeekCloser
	Size     int64
}

func TorrentFileDownloadController(user *engine.User, torrent_id string, index_str string, db *gorm.DB) (*FileReturn, error) {
	torrentId, err := strconv.Atoi(torrent_id)
	if err != nil {
		return nil, err
	}
	torrent := user.GetUserTorrent(uint(torrentId))
	if torrent == nil {
		return nil, errors.New("torrent not found")
	}
	index, err := strconv.Atoi(index_str)
	if err != nil {
		return nil, err
	}
	if index >= len(torrent.Torrent.Files()) || index < 0 {
		return nil, errors.New("file not found")
	}
	file := torrent.Torrent.Files()[index]
	file.Torrent().AllowDataDownload()

	file.SetPriority(types.PiecePriorityNow)
	fileName := path.Base(file.DisplayPath())
	return &FileReturn{fileName, file.NewReader(), file.Length()}, nil
}
func TorrentFileDownload(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	torrent_id := ctx.Query("id")
	index := ctx.Query("index")
	file, err := TorrentFileDownloadController(&user, torrent_id, index, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename="+file.FileName)
	kosmixutil.ServerRangeRequest(ctx, file.Size, file.Reader, true, true)
}
