package transcode

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

type ConvertPayload struct {
	FileId     int    `json:"file_id"`
	QualityRes int    `json:"quality_res"`
	AudioTrack int    `json:"audio_track_index"`
	Path       string `json:"path"`
}

func Convert(db *gorm.DB, ctx *gin.Context, app *gin.Engine) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	var payload ConvertPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid payload"})
		return
	}
	var file engine.FILE
	if tx := db.Where("id = ?", payload.FileId).First(&file); tx.Error != nil {
		ctx.JSON(404, gin.H{"error": "file not found"})
		return
	}
	ffprobeData, err := file.FfprobeData(app)
	if err != nil || ffprobeData == nil {
		ctx.JSON(500, gin.H{"error": "ffprobe failed"})
		return
	}
	if ffprobeData.AudioTrackByIndex(payload.AudioTrack) == nil {
		ctx.JSON(400, gin.H{"error": "invalid audio_track_index " + strconv.Itoa(len(ffprobeData.AudioStreams()))})
		return
	}
	storer, root_path_of, err := engine.ParsePath(payload.Path)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	storerPath, err := storer.DbElement.GetRootPath(root_path_of)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	qual := engine.GetQualityByResolution(payload.QualityRes)
	if qual == nil {
		ctx.JSON(400, gin.H{"error": "invalid quality_res"})
		return
	}
	task := user.CreateTask("Converting "+file.FILENAME, func() error { return errors.New("unimplemented") })
	convertItem := engine.Convert{
		SOURCE_FILE:     &file,
		OUTPUT_FILE:     nil,
		User:            &user,
		Start:           time.Now(),
		Task:            task,
		Quality:         qual,
		AudioTrackIndex: uint(payload.AudioTrack),
		FfprobeBase:     ffprobeData,

		Paused:           false,
		OutputPathStorer: storerPath,
		Progress: &engine.FfmpegProgress{
			Frame:         0,
			Fps:           0,
			Stream_0_0_q:  0,
			Bitrate:       0,
			Total_size:    0,
			Out_time_us:   0,
			Out_time_ms:   0,
			Out_time:      "0",
			Progress:      "0",
			Dup_frames:    0,
			Drop_frames:   0,
			Speed:         0.0,
			TotalProgress: 0,
		},
	}
	go convertItem.Convert(app)
	ctx.JSON(200, gin.H{"status": "success", "task_id": task.ID})

}
func ConvertOptions(db *gorm.DB, ctx *gin.Context, app *gin.Engine) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	fileId := ctx.Query("file_id")
	fileIdInt, err := strconv.Atoi(fileId)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid file_id"})
		return
	}
	var file engine.FILE
	if tx := db.Where("id = ?", fileIdInt).First(&file); tx.Error != nil {
		ctx.JSON(404, gin.H{"error": "file not found"})
		return
	}
	ffprobeData, err := file.FfprobeData(app)
	if err != nil || ffprobeData == nil {
		ctx.JSON(500, gin.H{"error": "ffprobe failed"})
		return
	}
	var Tracks []engine.AUDIO_TRACK = make([]engine.AUDIO_TRACK, 0)
	for _, stream := range ffprobeData.AudioStreams() {
		Tracks = append(Tracks, engine.AUDIO_TRACK{
			Index: stream.Index,
			Name:  stream.Tags.Language,
		})
	}
	availablePaths, err := engine.GetStorageRenders()
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{
		"Qualities":   engine.Config.Transcoder.Qualitys,
		"AudioTracks": Tracks,
		"Paths":       availablePaths,
	})
}
func Action(db *gorm.DB, ctx *gin.Context, app *gin.Engine) {
	action := ctx.Query("action")
	convertId, err := strconv.Atoi(ctx.Query("convert_id"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid convert_id"})
		return
	}
	if convertId <= 0 {
		ctx.JSON(400, gin.H{"error": "invalid convert_id"})
		return
	}
	convert := engine.GetConvertByTaskId(uint(convertId))
	if convert == nil {
		ctx.JSON(404, gin.H{"error": "convert not found"})
		return
	}
	switch action {
	case "resume":
		err = convert.Resume()
	case "pause":
		err = convert.Pause()

	case "stop":
		err = convert.Stop()
	default:
		ctx.JSON(404, gin.H{"error": "not found"})
		return
	}

	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}
