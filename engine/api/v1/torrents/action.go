package torrents

import (
	"errors"
	"strconv"

	"github.com/anacrolix/torrent/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func TorrentActionController(user *engine.User, torrent_id string, action string, deleteFiles string, fileIndex string, priority string, db *gorm.DB) error {

	torrentId, err := strconv.Atoi(torrent_id)
	if err != nil {
		return err
	}
	torrent := user.GetUserTorrent(uint(torrentId))
	if torrent == nil {
		return errors.New("torrent not found")
	}
	var torrentDbItem *engine.Torrent
	if err := db.Where("id = ?", torrent.DB_ITEM.ID).First(&torrentDbItem).Error; err != nil {
		return err
	}
	switch action {
	case "pause":
		if torrentDbItem.Paused {
			return errors.New("torrent already paused")
		}
		torrent.Torrent.DisallowDataDownload()
		torrent.Torrent.DisallowDataUpload()
		torrentDbItem.Paused = true
		if tx := db.Save(&torrentDbItem); tx.Error != nil {
			return tx.Error
		}
	case "resume":
		if !torrentDbItem.Paused {
			return errors.New("torrent already resumed")
		}
		torrent.Torrent.AllowDataDownload()
		torrent.Torrent.AllowDataUpload()
		torrentDbItem.Paused = false
		if tx := db.Save(&torrentDbItem); tx.Error != nil {
			return tx.Error
		}
	case "delete":
		if !user.ADMIN {
			return errors.New("not allowed")
		}
		if err := engine.CleanDeleteTorrent(deleteFiles == "true", torrent, db); err != nil {
			return err
		}
	case "recheck":
		if !user.ADMIN {
			return errors.New("not allowed")
		}
		go torrent.Torrent.VerifyData()
	case "download":
		if !user.ADMIN {
			return errors.New("not allowed")
		}
		for _, file := range torrent.Torrent.Files() {
			file.Download()
		}
		torrent.Torrent.DownloadAll()
	case "reannounce":
		if !user.ADMIN {
			return errors.New("not allowed")
		}
		// torrent.Torrent.UseSources()
	case "priority":
		if !user.ADMIN {
			return errors.New("not allowed")
		}
		fileIndex, err := strconv.Atoi(fileIndex)
		if err != nil {
			return err
		}
		priority, err := strconv.Atoi(priority)
		if err != nil {
			return err
		}
		if fileIndex < 0 || fileIndex >= len(torrent.Torrent.Files()) {
			return errors.New("invalid file index")
		}
		if priority < 0 || priority > 3 {
			return errors.New("invalid priority")
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
			return errors.New("invalid priority")
		}
	default:
		return errors.New("invalid action")
	}
	return nil
}
func TorrentsAction(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	torrent_id := ctx.Query("id")
	action := ctx.Query("action")
	deleteFiles := ctx.Query("deleteFiles")
	fileIndex := ctx.Query("fileIndex")
	priority := ctx.Query("priority")
	if err := TorrentActionController(&user, torrent_id, action, deleteFiles, fileIndex, priority, db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"success": true})
}
func TorrentsActionWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"torrent_id", "action", "deleteFiles", "fileIndex", "priority"}, request.Options)
	if err := TorrentActionController(&user, keys["torrent_id"], keys["action"], keys["deleteFiles"], keys["fileIndex"], keys["priority"], db); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"success": true}, nil, request.RequestUuid)
}
