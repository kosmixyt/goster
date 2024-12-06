package iptv

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

type AddRecordPayload struct {
	ChannelId    int64  `json:"channel_id"`
	Start        int64  `json:"start"`
	Duration     int64  `json:"duration"`
	OutputType   string `json:"output_type"`
	OutputId     int64  `json:"output_id"`
	Force        bool   `json:"force"`
	StorerOutput string `json:"storer_output"`
}

func AddRecord(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.CAN_TRANSCODE {
		ctx.JSON(401, gin.H{"error": "not allowed to record"})
		return
	}
	var payload AddRecordPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid payload"})
		return
	}
	task := user.CreateTask("Record IPTV", func() error { panic("unimplementd") })
	userChannel := user.GetUserChannel(int(payload.ChannelId))
	if userChannel == nil {
		task.SetAsError("channel not found")
		ctx.JSON(400, gin.H{"error": "channel not found"})
		return
	}
	var episode engine.EPISODE
	var movie engine.MOVIE
	if payload.OutputType == "episode" {
		if db.Where("id = ?", payload.OutputId).First(&episode).Error != nil {
			task.SetAsError("episode not found")
			ctx.JSON(400, gin.H{"error": "episode not found"})
			return
		}
	} else if payload.OutputType == "movie" {
		if db.Where("id = ?", payload.OutputId).First(&movie).Error != nil {
			task.SetAsError("movie not found")
			ctx.JSON(400, gin.H{"error": "movie not found"})
			return
		}
	} else {
		task.SetAsError("output_type must be episode or movie")
		ctx.JSON(400, gin.H{"error": "output_type must be episode or movie"})
		return
	}
	storer, path, err := engine.ParsePath(payload.StorerOutput)
	if err != nil {
		task.SetAsError(err)
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	record := engine.Record{
		START:                time.Unix(0, payload.Start*int64(time.Millisecond)),
		DURATION:             payload.Duration,
		OWNER_ID:             user.ID,
		ENDED:                false,
		IPTV_ID:              uint(userChannel.Iptv.ID),
		TASK_ID:              task.ID,
		CHANNEL_ID:           payload.ChannelId,
		Force:                payload.Force,
		OutputStorer:         storer.DbElement,
		OutputStorerMem:      storer,
		OutputStorerRootPath: path,
		OutputStorerId:       storer.DbElement.ID,
		OutputStorerFileName: "output.mp4",
		ERROR:                "",
	}
	if payload.OutputType == "episode" {
		record.OUTPUT_EPISODE_ID = &episode.ID
	}
	if payload.OutputType == "movie" {
		record.OUTPUT_MOVIE_ID = &movie.ID
	}
	db.Preload("OWNER").Save(&record)
	go record.Init()
	ctx.JSON(200, gin.H{"record": record})
}

func RemoveRecord(ctx *gin.Context, db *gorm.DB) {
	_, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	record_id := ctx.PostForm("record_id")
	recordId, err := strconv.ParseInt(record_id, 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "record_id must be an integer"})
		return
	}
	var record engine.Record
	if db.Where("id = ?", recordId).First(&record).Error != nil {
		ctx.JSON(400, gin.H{"error": "record not found"})
		return
	}
	db.Delete(&record)
	ctx.JSON(200, gin.H{"record": record})
}
