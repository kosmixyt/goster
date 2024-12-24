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
	"github.com/gorilla/websocket"
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
func NewTranscoderController(
	app *gin.Engine,
	db *gorm.DB,
	user *engine.User,
	progress func(event string, data string),
	fileIdstr string,
	mediaType string,
	season_str string,
	episode_str string,
	id string,
	torrent_id_str string,
	close_chan *(func() <-chan bool),
) {
	if !user.CAN_TRANSCODE_FILE() {
		progress(engine.ServerSideError, "not allowed to transcode")
		return
	}
	progress("progress", "Processing request")
	task := user.CreateTask("Transcode", func() error { return errors.New("uncancellable task") })
	var fileItem *engine.FILE
	if fileIdstr != "" {
		fileId, err := strconv.Atoi(fileIdstr)
		if err != nil {
			task.SetAsError(err)
			progress(engine.ServerSideError, "invalid fileId")
			return
		}
		var file engine.FILE
		if tx := db.Preload("MOVIE").Preload("TV").Where("id = ?", fileId).First(&file); tx.Error != nil {
			progress(engine.ServerSideError, task.SetAsError(tx.Error).(error).Error())
			return
		}
		progress("progress", "File found "+file.FILENAME)
		fileItem = &file
	} else {
		if mediaType != engine.Tv && mediaType != "movie" {
			progress(engine.ServerSideError, task.SetAsError(errors.New("invalid type")).(error).Error())
			return
		}
		provider, id, err := engine.ParseIdProvider(id)
		if err != nil {
			progress(engine.ServerSideError, task.SetAsError(err).(error).Error())
			return
		}
		switch provider {
		case "tmdb":
			provider = "tmdb_id"
		case "db":
			provider = "id"
		default:
			task.SetAsError(errors.New("invalid provider"))
			progress(engine.ServerSideError, "invalid provider")
			return
		}
		var season int = 0
		var episode int = 0
		if mediaType == engine.Tv {
			if _, err = fmt.Sscanf(season_str, "%d", &season); err != nil {
				task.SetAsError(err)
				progress(engine.ServerSideError, "invalid season")
				return
			}
			if _, err = fmt.Sscanf(episode_str, "%d", &episode); err != nil {
				task.SetAsError(err)
				progress(engine.ServerSideError, "invalid episode")
				return
			}
			if season == 0 || episode == 0 {
				task.SetAsError(errors.New("invalid season or episode"))
				progress(engine.ServerSideError, "invalid season or episode")
				return
			}
		}
		ItemFile, err := engine.GetMediaReader(db, user, mediaType, provider, id, season, episode, torrent_id_str, task, func(s string) { progress("progress", s) })
		if err != nil {
			progress(engine.ServerSideError, task.SetAsError(err).(error).Error())
			return
		}
		fileItem = ItemFile
	}
	if fileItem.ID == 0 {
		progress(engine.ServerSideError, task.SetAsError(errors.New("file not found")).(error).Error())
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
			Preload("TV.SEASON.EPISODES.WATCHING", "user_id = ?", user.ID).
			Preload("TV.SEASON.EPISODES.FILES")
	}
	var WatchingItem *engine.WATCHING = fileItem.GetWatching(user, preload)
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
		progress("progress", "Transcoder Waiting ffprobe Data")
		transcoder.GetData((*close_chan)())
		progress("progress", "Transcoder Data Received")
		WatchingItem.TOTAL = int64(transcoder.LENGTH)
		engine.Transcoders = append(engine.Transcoders, transcoder)
		Tracks = transcoder.TRACKS
		Subtitles = transcoder.SUBTITLES
		Url = transcoder.ManifestUrl()
		Uuid = transcoder.UUID
		Qu = transcoder.QUALITYS
	} else {
		Uuid = "--no-needed--"
		Url = engine.Create206Allowed(app, fileItem, user)
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
		res.Poster = WatchingItem.MOVIE.Poster("high")
		res.Backdrop = WatchingItem.MOVIE.Backdrop("high")
	} else {
		res.Seasons = WatchingItem.TV.ToSeason()
		res.Name = WatchingItem.TV.NAME + " S" + WatchingItem.EPISODE.SEASON.GetNumberAsString(true) + " E" + WatchingItem.EPISODE.GetNumberAsString(true)
		res.Poster = WatchingItem.TV.Poster("high")
		res.Backdrop = WatchingItem.TV.Backdrop("high")
	}
	b, err := json.Marshal(res)
	if err != nil {
		progress(engine.ServerSideError, task.SetAsError(err).(error).Error())
		return
	}

	progress("transcoder", string(b))
}

func NewTranscoder(app *gin.Engine, ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		kosmixutil.SendEvent(ctx, engine.ServerSideError, "not logged in")
		return
	}
	f := func() <-chan bool { return ctx.Writer.CloseNotify() }
	NewTranscoderController(
		app,
		db,
		&user,
		func(event, data string) {
			kosmixutil.SendEvent(ctx, event, data)
		},
		ctx.Query("fileId"),
		ctx.Query("type"),
		ctx.Query("season"),
		ctx.Query("episode"),
		ctx.Query("id"),
		ctx.Query("torrent_id"),
		&f,
	)
}
func NewTranscoderWs(app *gin.Engine, db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKeys([]string{"fileId", "type", "season", "episode", "id", "torrent_id"}, request.Options)
	NewTranscoderController(
		app,
		db,
		&user,
		func(event, data string) { fmt.Println("Event", event, data) },
		key["fileId"],
		key["type"],
		key["season"],
		key["episode"],
		key["id"],
		key["torrent_id"],
		nil,
	)
	kosmixutil.SendWebsocketResponse(conn, gin.H{"success": true}, nil, request.RequestUuid)
}

func TranscodeSegment(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	number := ctx.Param("number")
	quality := ctx.Request.Header.Get("X-Quality")
	x_track := ctx.Request.Header.Get("X-Track")
	currentPlayBack := ctx.Request.Header.Get("X-Current-Time")
	data, err := TranscodeSegmentController(&user, uuid, number, quality, x_track, currentPlayBack)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.Data(200, "application/octet-stream", data)
}
func TranscodeSegmentController(user *engine.User, uuid string, number string, QualityName string, x_track string, currentPlayBack string) ([]byte, error) {

	transcoder := user.GetTranscode(uuid)
	if transcoder == nil {
		return nil, errors.New("transcoder not found")
	}
	var index int
	if _, err := fmt.Sscanf(number, "%d", &index); err != nil {
		return nil, errors.New("invalid number")
	}
	if index < 0 {
		return nil, errors.New("invalid number")
	}
	if QualityName == "" {
		QualityName = "1080p"
	}
	TrackIndex := 0
	if _, err := fmt.Sscanf(x_track, "%d", &TrackIndex); err != nil {
		return nil, errors.New("invalid track")
	}
	if !transcoder.HasAudioStream(TrackIndex) {
		return nil, errors.New("invalid track")
	}
	qual, err := transcoder.GetQuality(QualityName)
	if err != nil {
		return nil, errors.New("invalid quality")
	}
	if currentPlayBack != "" {
		var current int64 = 0
		if _, err = fmt.Sscanf(currentPlayBack, "%d", &current); err != nil {
			return nil, errors.New("invalid current")
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
		return nil, errors.New("error while getting segment")
	}
	data, err := io.ReadAll(segment)
	if err != nil {
		return nil, errors.New("error while reading segment")
	}
	// ctx.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	// ctx.Data(200, "application/octet-stream", data)
	return data, nil
}
func TranscodeSegmentWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"uuid", "number", "quality", "track", "current"}, request.Options)
	data, err := TranscodeSegmentController(&user, keys["uuid"], keys["number"], keys["quality"], keys["track"], keys["current"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}

	for i := 0; i < len(data)/1024; i++ {
		kosmixutil.SendWebsocketResponse(conn, data[i*1024:(i+1)*1024], nil, request.RequestUuid)
	}
	kosmixutil.SendWebsocketResponse(conn, "", nil, request.RequestUuid)

}
func TranscodeManifest(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	manifest, err := TranscodeManifestController(&user, uuid)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.Data(200, "text/plain", manifest)
}
func TranscodeManifestController(user *engine.User, uuid string) ([]byte, error) {
	transcoder := user.GetTranscode(uuid)
	if transcoder == nil {
		return nil, errors.New("transcoder not found")
	}
	channel := make(chan string)
	go transcoder.Manifest(channel)
	manifest := <-channel
	return []byte(manifest), nil
}
func TranscodeManifestWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKey("uuid", request.Options)
	manifest, err := TranscodeManifestController(&user, key)
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, manifest, nil, request.RequestUuid)
}
func TranscodeSubtitle(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := ctx.Param("uuid")
	index := ctx.Param("index")
	reader, err := TranscodeSubtitleController(&user, uuid, index)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.DataFromReader(200, -1, "text/vtt", reader, nil)
}

func TranscodeSubtitleController(user *engine.User, uuid string, index string) (io.Reader, error) {

	number := index
	var intIndex int
	if _, err := fmt.Sscanf(number, "%d", &intIndex); err != nil {
		return nil, errors.New("invalid number")
	}
	transcoder := user.GetTranscode(uuid)
	if transcoder == nil {
		return nil, errors.New("transcoder not found")
	}
	if transcoder.ISLIVESTREAM {
		return nil, errors.New("live stream")
	}
	result := make(chan io.Reader, 1)
	go transcoder.GetSubtitle(intIndex, result)
	reader := <-result
	if reader == nil {
		return nil, errors.New("subtitle not found")
	}
	return reader, nil
}
func TranscodeSubtitleWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"uuid", "index"}, request.Options)
	reader, err := TranscodeSubtitleController(&user, keys["uuid"], keys["index"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, reader, nil, request.RequestUuid)
}

// watching -> Preload movie, Episode, tv, tv.season, tv.season.episodes, user, episode.season
// must preload -> movie.genre; tv.genre; tv.seasons
