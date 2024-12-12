package iptv

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
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

func AddRecordController(user *engine.User, payload AddRecordPayload, db *gorm.DB) error {
	if !user.CAN_TRANSCODE {
		return engine.ErrorCannotRecord
	}

	task := user.CreateTask("Record IPTV", func() error { panic("unimplementd") })
	userChannel := user.GetUserChannel(int(payload.ChannelId))
	if userChannel == nil {
		task.SetAsError("channel not found")
		return engine.ErrorChannelNotFound
	}
	var episode engine.EPISODE
	var movie engine.MOVIE
	if payload.OutputType == "episode" {
		if db.Where("id = ?", payload.OutputId).First(&episode).Error != nil {
			task.SetAsError("episode not found")
			return engine.ErrorEpisodeNotFound
		}
	} else if payload.OutputType == "movie" {
		if db.Where("id = ?", payload.OutputId).First(&movie).Error != nil {
			task.SetAsError("movie not found")
			return engine.ErrorMovieNotFound
		}
	} else {
		task.SetAsError("output_type must be episode or movie")
		// ctx.JSON(400, gin.H{"error": "output_type must be episode or movie"})
		return engine.ErrorInvalidOutputType
	}
	storer, path, err := engine.ParsePath(payload.StorerOutput)
	if err != nil {
		task.SetAsError(err)
		return err
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
	return nil
}

func AddRecord(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var payload AddRecordPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := AddRecordController(&user, payload, db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
}

func AddRecordWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	payload, ok := request.Options.(AddRecordPayload)
	if !ok {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("bad payload"), request.RequestUuid)
		return
	}
	if err := AddRecordController(&user, payload, db); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"status": "success"}, nil, request.RequestUuid)
}
