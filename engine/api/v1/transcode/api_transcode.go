package transcode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Charset", "utf-8")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Flush()
		c.Next()
	}
}
func NewTranscoder(app *gin.Engine, ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		kosmixutil.SendEvent(ctx, engine.ServerSideError, "not logged in")
		return
	}
	if !user.CAN_TRANSCODE_FILE() {
		kosmixutil.SendEvent(ctx, engine.ServerSideError, "not allowed to transcode")
		return
	}
	kosmixutil.SendEvent(ctx, "progress", "Processing request")
	task := user.CreateTask("Transcode", func() error { return errors.New("uncancellable task") })
	fileIdstr := ctx.Query("fileId")
	var fileItem *engine.FILE
	if fileIdstr != "" {
		fileId, err := strconv.Atoi(fileIdstr)
		if err != nil {
			task.SetAsError(err)
			kosmixutil.SendEvent(ctx, engine.ServerSideError, "invalid fileId")
			return
		}
		var file engine.FILE
		if tx := db.Preload("MOVIE").Preload("TV").Where("id = ?", fileId).First(&file); tx.Error != nil {
			kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(tx.Error).(error).Error())
			return
		}
		kosmixutil.SendEvent(ctx, "progress", "File found "+file.FILENAME)
		fileItem = &file
	} else {
		var mediaType = ctx.Query("type")
		if mediaType != engine.Tv && mediaType != "movie" {
			kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(errors.New("invalid type")).(error).Error())
			return
		}
		provider, id, err := engine.ParseIdProvider(ctx.Query("id"))
		if err != nil {
			kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(err).(error).Error())
			return
		}
		switch provider {
		case "tmdb":
			provider = "tmdb_id"
		case "db":
			provider = "id"
		default:
			task.SetAsError(errors.New("invalid provider"))
			kosmixutil.SendEvent(ctx, engine.ServerSideError, "invalid provider")
			return
		}
		var season int = 0
		var episode int = 0
		if mediaType == engine.Tv {
			if _, err = fmt.Sscanf(ctx.Query("season"), "%d", &season); err != nil {
				task.SetAsError(err)
				kosmixutil.SendEvent(ctx, engine.ServerSideError, "invalid season")
				return
			}
			if _, err = fmt.Sscanf(ctx.Query("episode"), "%d", &episode); err != nil {
				task.SetAsError(err)
				kosmixutil.SendEvent(ctx, engine.ServerSideError, "invalid episode")
				return
			}
			if season == 0 || episode == 0 {
				task.SetAsError(errors.New("invalid season or episode"))
				kosmixutil.SendEvent(ctx, engine.ServerSideError, "invalid season or episode")
				return
			}
		}
		torrent_id_str := ctx.Query("torrent_id")
		ItemFile, err := engine.GetMediaReader(db, &user, mediaType, provider, id, season, episode, torrent_id_str, task, func(s string) { kosmixutil.SendEvent(ctx, "progress", s) })
		if err != nil {
			kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(err).(error).Error())
			return
		}
		fileItem = ItemFile
	}
	if fileItem.ID == 0 {
		kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(errors.New("file not found")).(error).Error())
		return
	}
	preload := func() *gorm.DB {
		return db.Preload("USER").
			Preload("MOVIE").
			Preload("MOVIE.GENRE").
			Preload("EPISODE").
			Preload("EPISODE.SEASON").
			Preload("TV").
			Preload("TV.GENRE").
			Preload("TV.SEASON").
			Preload("TV.SEASON.EPISODES").
			Preload("TV.SEASON.EPISODES.WATCHING", "user_id = ?", user.ID)
	}
	var WatchingItem *engine.WATCHING = fileItem.GetWatching(&user, preload)
	db.Save(&user)
	Tracks := make([]engine.AUDIO_TRACK, 1)
	Subtitles := make([]engine.SUBTITLE, 1)
	Qu := make([]engine.QUALITY, 1)
	var Url string
	var Uuid string
	if engine.Config.Transcoder.EnableForWebPlayableFiles || !fileItem.IsBrowserPlayable() {
		ffUrl := fileItem.Ffurl(app)
		on_progress := func(current int64, total int64) {
			db.Model(&engine.WATCHING{}).Where("id = ?", WatchingItem.ID).Update("current", current).Update(
				"total", total,
			)
		}
		on_destroy := func(t *engine.Transcoder) {
			index := -1
			for i, transcoder := range engine.Transcoders {
				if transcoder.UUID == t.UUID {
					index = i
					break
				}
			}
			if index == -1 {
				panic("Transcoder not found")
			}
			engine.Transcoders = append(engine.Transcoders[:index], engine.Transcoders[index+1:]...)
		}
		ct := time.Now()
		task.SetAsStarted()
		transcoder := &engine.Transcoder{
			CURRENT_QUALITY:   nil,
			ON_REQ_TIMEOUT:    func(t *engine.Transcoder) {},
			ON_DESTROY:        on_destroy,
			UUID:              uuid.New().String(),
			OWNER_ID:          user.ID,
			FFPROBE_TIMEOUT:   5 * time.Second,
			ISLIVESTREAM:      false,
			FFURL:             ffUrl,
			ON_PROGRESS:       on_progress,
			SEGMENTS:          make(map[string](chan (func() io.Reader))),
			Last_request_time: &ct,
			Task:              task,
			TRACKS:            make([]engine.AUDIO_TRACK, 0),
			SUBTITLES:         make([]engine.SUBTITLE, 0),
			CURRENT_TRACK:     -1,
			LENGTH:            -1,
			QUALITYS:          make([]engine.QUALITY, 0),
			CHUNK_LENGTH:      -1,
			Ffprobe:           nil,
			Request_pending:   false,
			CURRENT_INDEX:     -1,
			START_INDEX:       -1,
			Source:            fileItem,
			FFMPEG:            nil,
			FETCHING_DATA:     false,
			Db:                db,
		}
		kosmixutil.SendEvent(ctx, "progress", "Transcoder Waiting ffprobe Data")
		transcoder.GetData(ctx.Writer.CloseNotify())
		kosmixutil.SendEvent(ctx, "progress", "Transcoder Data Received")
		WatchingItem.TOTAL = int64(transcoder.LENGTH)
		engine.Transcoders = append(engine.Transcoders, transcoder)
		Tracks = transcoder.TRACKS
		Subtitles = transcoder.SUBTITLES
		Url = transcoder.ManifestUrl()
		Uuid = transcoder.UUID
		Qu = transcoder.QUALITYS
	} else {
		Uuid = "--no-needed--"
		Url = engine.Create206Allowed(app, fileItem, &user)
	}

	res := engine.TranscoderRes{
		Manifest:          Url,
		Uuid:              Uuid,
		Task_id:           task.ID,
		Qualitys:          Qu,
		Download_url:      fileItem.GetDownloadUrl(),
		Tracks:            Tracks,
		Subtitles:         Subtitles,
		Current:           WatchingItem.CURRENT,
		Total:             WatchingItem.TOTAL,
		Name:              fileItem.FILENAME,
		Poster:            "https://image.tmdb.org/t/p/original/nHo5pZIIgAq2a9SJOUkfmldQHcB.jpg",
		Backdrop:          "https://image.tmdb.org/t/p/original/zYTUdAeUwHZLFqZXvMLhd1oOszh.jpg",
		Next:              *WatchingItem.GetNextFile(),
		IsLive:            false,
		IsBrowserPlayable: fileItem.IsBrowserPlayable() && !engine.Config.Transcoder.EnableForWebPlayableFiles,
	}
	if WatchingItem.MOVIE_ID != 0 {
		res.Name = WatchingItem.MOVIE.NAME
		res.Poster = WatchingItem.MOVIE.Poster(engine.TMDB_ORIGINAL)
		res.Backdrop = WatchingItem.MOVIE.Backdrop(engine.TMDB_ORIGINAL)
	} else {
		res.Seasons = WatchingItem.TV.ToSeason()
		res.Name = WatchingItem.TV.NAME + " " + WatchingItem.EPISODE.SEASON.GetNumberAsString(true) + " " + WatchingItem.EPISODE.GetNumberAsString(true)
		res.Poster = WatchingItem.TV.Poster(engine.TMDB_ORIGINAL)
		res.Backdrop = WatchingItem.TV.Backdrop(engine.TMDB_ORIGINAL)
	}
	b, err := json.Marshal(res)
	if err != nil {
		kosmixutil.SendEvent(ctx, engine.ServerSideError, task.SetAsError(err).(error).Error())
		return
	}

	kosmixutil.SendEvent(ctx, "transcoder", string(b))
}

func NewTranscoderController() {

}

func TranscodeSegment(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	number := ctx.Param("number")
	var transcoder *engine.Transcoder
	for _, t := range engine.Transcoders {
		if t.UUID == uuid && user.ID == t.OWNER_ID {
			transcoder = t
			break
		}
	}
	if transcoder == nil {
		ctx.JSON(404, gin.H{"error": "transcoder not found"})
		return
	}

	var index int
	if _, err = fmt.Sscanf(number, "%d", &index); err != nil {
		fmt.Println("error while scanning number", number)
		ctx.JSON(400, gin.H{"error": "invalid number"})
		return
	}
	if index < 0 {
		ctx.JSON(400, gin.H{"error": "invalid number"})
		return
	}
	QualityName := ctx.Request.Header.Get("X-QUALITY")
	if QualityName == "" {
		QualityName = "1080p"
	}
	TrackIndex := 0
	if _, err = fmt.Sscanf(ctx.Request.Header.Get("X-TRACK"), "%d", &TrackIndex); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid track"})
		return
	}
	if !transcoder.HasAudioStream(TrackIndex) {
		ctx.JSON(400, gin.H{"error": "invalid track"})
		return
	}
	qual, err := transcoder.GetQuality(QualityName)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid quality"})
		return
	}
	// if (*transcoder.Last_request_time).Add(100 * time.Millisecond).After(time.Now()) {
	// 	ctx.JSON(429, gin.H{"error": "too many requests"})
	// 	return
	// }
	currentPlayBack := ctx.Request.Header.Get("X-CURRENT-TIME")
	if currentPlayBack != "" {
		var current int64 = 0
		if _, err = fmt.Sscanf(currentPlayBack, "%d", &current); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid current time"})
			return
		}
		transcoder.SetCurrentTime(current, index)
	}
	transcoder.Request_pending = true
	ct := time.Now()
	transcoder.Last_request_time = &ct
	segment, err := transcoder.Chunk(index, qual, TrackIndex)
	transcoder.Request_pending = false
	if err != nil {
		transcoder.Task.AddLog("Error while getting segment: " + err.Error())
		ctx.JSON(500, gin.H{"error": "error while getting segment" + err.Error()})
		return
	}
	data, err := io.ReadAll(segment)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "error while reading segment" + err.Error()})
		return
	}
	ctx.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	ctx.Data(200, "application/octet-stream", data)
	data = nil
}
func TranscodeManifest(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	var transcoder *engine.Transcoder
	for _, tr := range engine.Transcoders {
		if tr.UUID == uuid && user.ID == tr.OWNER_ID {
			transcoder = tr
			break
		}
	}
	if transcoder == nil {
		ctx.JSON(404, gin.H{"error": "transcoder not found"})
		return
	}
	channel := make(chan string)
	go transcoder.Manifest(channel)
	ctx.Data(200, "application/x-mpegURL", []byte(<-channel))
}

func TranscodeSubtitle(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	fmt.Println("-------- SUBTITLE ---------", ctx.Param("uuid"))
	number := ctx.Param("index")
	var intIndex int
	if _, err := fmt.Sscanf(number, "%d", &intIndex); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid subtitle index"})
		return
	}
	transcoder := user.GetTranscode(ctx.Param("uuid"))
	if transcoder == nil {
		ctx.JSON(
			404,
			gin.H{"error": "transcoder not found"},
		)
		return
	}
	if transcoder.ISLIVESTREAM {
		ctx.JSON(404, gin.H{"error": "subtitle not found"})
		return
	}
	result := make(chan io.Reader, 1)
	go transcoder.GetSubtitle(intIndex, result)
	reader := <-result
	if reader == nil {
		ctx.JSON(404, gin.H{"error": "subtitle not found"})
		return
	}
	ctx.DataFromReader(200, -1, "text/vtt", reader, map[string]string{})
}

// watching -> Preload movie, Episode, tv, tv.season, tv.season.episodes, user, episode.season
// must preload -> movie.genre; tv.genre; tv.seasons

type EpiNumber struct {
	SEASON  int
	EPISODE int
}
