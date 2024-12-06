package download

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func DownloadItem(ctx *gin.Context, db *gorm.DB) {
	reqTime := time.Now()
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	renderFile, task, err := DownloadItemController(ctx.Query("fileId"), &user, db, ctx.Query("type"), ctx.Query("season"), ctx.Query("episode"), ctx.Query("id"), ctx.Query("torrent_id"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	task.SetAsStarted()
	task.AddLog("Download started")
	task.AddLog("Download time: ", time.Since(reqTime).String())
	ctx.Header("Content-Disposition", "attachment; filename="+kosmixutil.FormatFilenameForContentDisposition(renderFile.FILENAME))
	ctx.Header("X-File-ID", strconv.Itoa(int(renderFile.ID)))
	if reader := renderFile.GetReader(); reader != nil {
		fmt.Println("Seekable reader", reader)
		kosmixutil.ServerRangeRequest(ctx, renderFile.SIZE, reader, true, true)
	} else {
		if Nreader := renderFile.GetNonSeekableReader(); Nreader != nil {
			fmt.Println("Non seekable reader")
			kosmixutil.ServerNonSeekable(ctx, Nreader)
		}
	}
	task.AddLog("Byte Transferred ", strconv.FormatInt(int64(ctx.Writer.Size()), 10))
	task.AddLog("Total download time: ", time.Since(reqTime).String())
	task.SetAsFinished()
}

func DownloadItemController(fileId_str string, user *engine.User, db *gorm.DB, itype string, season_str string, episode_str string, id string, torrent_id_str string) (*engine.FILE, *engine.Task, error) {
	var err error
	if !user.CAN_DOWNLOAD {
		return nil, nil, errors.New("not allowed to download")
	}
	var fileIdstr = fileId_str
	fileId := 0
	var renderFile engine.FILE
	task := user.CreateTask("Download - "+renderFile.FILENAME, func() error { return errors.New("un cancellable") })
	if fileIdstr != "" {
		fileId, err = strconv.Atoi(fileIdstr)
		if err != nil {
			return nil, nil, task.SetAsError(errors.New("invalid file id")).(error)
		}
		var file engine.FILE
		if tx := db.Where("id = ?", fileId).First(&file); tx.Error != nil {
			return nil, nil, task.SetAsError(errors.New("file not found")).(error)
		}
		renderFile = file
	} else {
		var mediaType = itype
		if mediaType != engine.Tv && mediaType != engine.Movie {
			return nil, nil, task.SetAsError(errors.New("invalid type")).(error)
		}
		provider, id, err := engine.ParseIdProvider(id)
		if err != nil {
			return nil, nil, task.SetAsError(err).(error)
		}
		if provider != "db" && provider != "tmdb" {
			return nil, nil, task.SetAsError(errors.New("invalid provider")).(error)
		}
		if provider == "tmdb" {
			provider = "tmdb_id"
		} else if provider == "db" {
			provider = "id"
		} else {
			return nil, nil, task.SetAsError(errors.New("invalid provider")).(error)
		}
		var file *engine.FILE
		if mediaType == engine.Tv {
			var seasonstr = season_str
			var season int
			_, err = fmt.Sscanf(seasonstr, "%d", &season)
			if err != nil {
				return nil, nil, task.SetAsError(errors.New("invalid season")).(error)
			}
			var episodestr = episode_str
			var episode int
			_, err = fmt.Sscanf(episodestr, "%d", &episode)
			if err != nil {
				return nil, nil, task.SetAsError(errors.New("invalid episode")).(error)
			}
			tempFile, err := engine.GetMediaReader(db, user, mediaType, provider, id, season, episode, torrent_id_str, task, func(s string) {})
			file = tempFile
			if err != nil {
				return nil, nil, task.SetAsError(err).(error)
			}
		} else {
			var tempFile *engine.FILE
			tempFile, err = engine.GetMediaReader(db, user, mediaType, provider, id, 0, 0, torrent_id_str, task, func(s string) {})
			if err != nil {
				return nil, nil, task.SetAsError(err).(error)
			}

			file = tempFile
		}

		renderFile = *file
		if mediaType == engine.Tv {
			if tx := db.Table("tvs").Where(provider+" = ?", id).Update("download", gorm.Expr("download + 1")); tx.Error != nil {
				panic("cannot update view of tv show")
			}
		} else if mediaType == engine.Movie {
			if tx := db.Table("movies").Where(provider+" = ?", id).Update("download", gorm.Expr("download + 1")); tx.Error != nil {
				fmt.Println(tx.Error)
				panic("cannot update view of movie")
			} else {
				fmt.Println("Movie view updated")
			}
		}
	}
	if renderFile.IsTorrentFile() {
		if renderFile.GetFileInTorrent().BytesCompleted() == 0 {
			return nil, nil, task.SetAsError(errors.New("file not ready")).(error)
		}
	}

	return &renderFile, task, nil
}
