package me

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func HandleMeController(user *engine.User, db *gorm.DB) (*Me, error) {

	var Requests []Me_request = make([]Me_request, 0)
	for _, request := range user.Requests {
		creq := Me_request{
			ID:          request.ID,
			Created:     request.CreatedAt,
			MaxSize:     uint(request.MAX_SIZE),
			Interval:    uint(request.Interval),
			Status:      request.STATUS,
			Last_Update: time.Since(request.LAST_TRY) / time.Second,
		}
		if request.TV != nil {
			creq.Render = request.TV.Skinny(nil)
			creq.Media_Name = request.TV.NAME
			creq.Media_Type = engine.Tv
			creq.Media_ID = request.TV.IdString()
		} else {
			creq.Render = request.Movie.Skinny(nil)
			creq.Media_Name = request.Movie.NAME
			creq.Media_Type = engine.Movie
			creq.Media_ID = request.Movie.IdString()
		}
		if request.TORRENT != nil {
			creq.Torrent_Name = request.TORRENT.Name
			creq.Torrent_ID = request.TORRENT.ID
		}
		Requests = append(Requests, creq)
	}
	var shares []Me_Share = make([]Me_Share, 0)
	for _, share := range user.SHARES {
		item := Me_Share{
			ID:     share.ID,
			EXPIRE: share.EXPIRE,
			FILE: engine.FileItem{
				DOWNLOAD_URL:  share.FILE.GetDownloadUrl(),
				FILENAME:      share.FILE.FILENAME,
				ID:            share.FILE.ID,
				SIZE:          share.FILE.SIZE,
				TRANSCODE_URL: share.FILE.GetTranscodeUrl(),
			},
			MEDIA_TYPE: share.FILE.GetMediaType(),
			MEDIA_ID:   "db@" + strconv.Itoa(share.FILE.GetMediaId()),
		}

		shares = append(shares, item)
	}
	var tl []tlItem = []tlItem{}
	torrent := user.GetTorrents()
	for _, t := range torrent {
		var torrentDbItem *engine.Torrent
		if err := db.Where("id = ?", t.DB_ID).
			Preload("FILES").
			Preload("FILES.TV").
			Preload("FILES.EPISODE").
			Preload("FILES.MOVIE").
			Preload("FILES.SEASON").
			First(&torrentDbItem).Error; err != nil {
			return nil, err
		}
		stats := t.Torrent.Stats()
		item := tlItem{
			ID:              torrentDbItem.ID,
			NAME:            torrentDbItem.Name,
			SIZE:            t.Torrent.Info().TotalLength(),
			PEERS:           int64(stats.ActivePeers),
			MAXPEERS:        int64(stats.TotalPeers),
			MediaOutput:     t.MEDIA_TYPE,
			MediaOutputUuid: "db@" + strconv.Itoa(int(t.MEDIA_UUID)),
			TotalDownloaded: torrentDbItem.DOWNLOAD,
			TotalUploaded:   torrentDbItem.UPLOAD,
			Progress:        torrentDbItem.Progress,
			FILES:           []SKINNY_FILES{},
			PAUSED:          torrentDbItem.Paused,
			Added:           torrentDbItem.CreatedAt.Unix(),
		}
		if item.MediaOutput == engine.Tv {
			for _, f := range torrentDbItem.FILES {
				if f.TV != nil && f.IS_MEDIA {
					item.SKINNY = f.TV.Skinny(nil)
					break
				}
			}
		} else {
			for _, f := range torrentDbItem.FILES {
				if f.MOVIE != nil && f.IS_MEDIA {
					item.SKINNY = f.MOVIE.Skinny(nil)
					break
				}
			}
		}
		if t.Torrent.Complete.Bool() {
			item.STATUS += "completed"
		} else {
			item.STATUS += "downloading"
		}

		if t.Torrent.Seeding() {
			item.STATUS += "|seeding"
		} else {

			item.STATUS += "|not seeding"
		}
		if torrentDbItem.Paused {
			item.STATUS += "|paused"
		} else {
			item.STATUS += "|not paused"
		}
		for _, f := range torrentDbItem.FILES {
			item.FILES = append(item.FILES, SKINNY_FILES{
				NAME: f.FILENAME,
				SIZE: f.SIZE,
				// useless (not show in the ui)
				PATH:     f.SUB_PATH,
				PROGRESS: float64(f.GetFileInTorrent().BytesCompleted()) / float64(f.SIZE),
				PRIORITY: int(f.GetFileInTorrent().Priority()),
			})
		}

		tl = append(tl, item)
	}
	converts := user.GetConverts()
	convertsRender := make([]ConvertRender, len(converts))
	for i, convert := range converts {
		convertsRender[i] = ConvertRender{
			SOURCE:          convert.SOURCE_FILE.SkinnyRender(user),
			Quality:         convert.Quality.Name,
			Task_id:         convert.Task.ID,
			AudioTrackIndex: convert.AudioTrackIndex,
			Running:         convert.Running,
			Paused:          convert.Paused,
			FILE: engine.FileItem{
				DOWNLOAD_URL:  convert.SOURCE_FILE.GetDownloadUrl(),
				FILENAME:      convert.SOURCE_FILE.FILENAME,
				ID:            convert.SOURCE_FILE.ID,
				SIZE:          convert.SOURCE_FILE.SIZE,
				TRANSCODE_URL: convert.SOURCE_FILE.GetTranscodeUrl(),
			},
			TaskStatus: convert.Task.Status,
			Progress:   *convert.Progress,
			Start:      convert.Start.Unix(),
		}
	}

	return &Me{
		ID:                  user.ID,
		Username:            user.NAME,
		Converts:            convertsRender,
		Torrents:            tl,
		Requests:            Requests,
		Shares:              shares,
		AllowedUploadNumber: uint(user.ALLOWED_UPLOAD_NUMBER),
		CurrentUploadNumber: uint(user.CURRENT_UPLOAD_NUMBER),
		AllowedUploadSize:   uint(user.ALLOWED_UPLOAD_SIZE),
		CurrentUploadSize:   uint(user.CURRENT_UPLOAD_SIZE),
		AllowedTranscode:    uint(user.MAX_TRANSCODING),
		CurrentTranscode:    uint(user.TRANSCODING),
	}, nil
}

func HandleMe(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{"SHARES", "SHARES.FILE", "Requests", "Requests.TV", "Requests.TV_SEASON", "Requests.Movie", "Requests.TORRENT"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if me, err := HandleMeController(&user, db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
	} else {
		ctx.JSON(200, me)
	}
}
func HandleMeWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{"SHARES", "SHARES.FILE", "Requests", "Requests.TV", "Requests.TV_SEASON", "Requests.Movie", "Requests.TORRENT"})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	if me, err := HandleMeController(&user, db); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
	} else {
		kosmixutil.SendWebsocketResponse(conn, me, nil, request.RequestUuid)
	}
}

type Me struct {
	ID                  uint            `json:"id"`
	Username            string          `json:"username"`
	Requests            []Me_request    `json:"requests"`
	AllowedUploadNumber uint            `json:"allowed_upload_number"`
	CurrentUploadNumber uint            `json:"current_upload_number"`
	AllowedUploadSize   uint            `json:"allowed_upload_size"`
	CurrentUploadSize   uint            `json:"current_upload_size"`
	AllowedTranscode    uint            `json:"allowed_transcode"`
	CurrentTranscode    uint            `json:"current_transcode"`
	Shares              []Me_Share      `json:"shares"`
	Converts            []ConvertRender `json:"converts"`
	Torrents            []tlItem
}
type Me_Noticiation struct {
	ID      uint
	Message string
}
type ConvertRender struct {
	FILE            engine.FileItem       `json:"file"`
	SOURCE          engine.SKINNY_RENDER  `json:"source"`
	Paused          bool                  `json:"paused"`
	Quality         string                `json:"quality"`
	Task_id         uint                  `json:"task_id"`
	AudioTrackIndex uint                  `json:"audio_track_index"`
	Running         bool                  `json:"running"`
	TaskStatus      string                `json:"task_status"`
	TaskError       string                `json:"task_error"`
	Progress        engine.FfmpegProgress `json:"progress"`
	Start           int64                 `json:"start"`
}
type SKINNY_FILES struct {
	NAME     string  `json:"name"`
	PATH     string  `json:"path"`
	SIZE     int64   `json:"size"`
	PROGRESS float64 `json:"progress"`
	PRIORITY int     `json:"priority"`
}

type Me_request struct {
	ID           uint
	Created      time.Time
	Type         string
	Last_Update  time.Duration
	MaxSize      uint
	Status       string
	Interval     uint
	Media_Name   string
	Media_Type   string
	Media_ID     string
	Torrent_ID   uint
	Torrent_Name string
	Render       engine.SKINNY_RENDER
}
type Me_Share struct {
	ID         uint
	EXPIRE     time.Time
	FILE       engine.FileItem
	MEDIA_TYPE string
	MEDIA_ID   string
}

type tlItem struct {
	ID              uint    `json:"id"`
	NAME            string  `json:"name"`
	SIZE            int64   `json:"size"`
	PEERS           int64   `json:"peers"`
	MAXPEERS        int64   `json:"maxpeers"`
	MediaOutput     string  `json:"mediaOutput"`
	TotalDownloaded int64   `json:"totalDownloaded"`
	TotalUploaded   int64   `json:"totalUploaded"`
	STATUS          string  `json:"status"`
	MediaOutputUuid string  `json:"mediaOutputUuid"`
	Progress        float64 `json:"progress"`
	Added           int64   `json:"added"`
	PAUSED          bool    `json:"paused"`
	SKINNY          engine.SKINNY_RENDER
	FILES           []SKINNY_FILES `json:"files"`
}
