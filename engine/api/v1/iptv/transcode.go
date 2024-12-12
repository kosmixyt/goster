package iptv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TranscodeIptv(ctx *gin.Context, db *gorm.DB) {
	SendEvent := func(event string, data string) {
		event = strings.ReplaceAll(event, "\n", "")
		data = strings.ReplaceAll(data, "\n", "")
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("Recover from error")
			}
		}()
		if ctx.Writer != nil {
			fmt.Fprintf(ctx.Writer, "event: %s\n", event)
			fmt.Fprintf(ctx.Writer, "data: %s\n\n", data)
			if ctx.Writer != nil {
				ctx.Writer.Flush()
			}
		}
	}
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		SendEvent(engine.ServerSideError, "Not logged in")
		return
	}
	fmt.Println("Got user")
	if err := TranscodeIptvController(db, &user, ctx.Query("channel"), SendEvent, ctx.Writer.CloseNotify()); err != nil {
		SendEvent(engine.ServerSideError, err.Error())
	}
}

func TranscodeIptvController(db *gorm.DB, user *engine.User, channelIdStr string, on_progress func(string, string), stop <-chan bool) error {
	task := user.CreateTask("Transcode IPTV", func() error { return errors.New("uncancellable task") })
	if !user.CAN_TRANSCODE {
		return errors.New("not allowed to transcode")
	}
	channelId, err := strconv.Atoi(channelIdStr)
	if err != nil {
		return err
	}
	channel := user.GetUserChannel(channelId)
	if channel == nil {
		return errors.New("Channel not found")
	}
	fmt.Println("Channel found")
	if channel.Iptv.CurrentStreamCount >=
		channel.Iptv.MaxStreamCount {
		return errors.New("Max stream count reached")
	}
	channel.Iptv.CurrentStreamCount += 1
	ct := time.Now()
	task.SetAsStarted()
	url := channel.Url[0 : len(channel.Url)-1]
	dataStream :=
		url
	transcoder := &engine.Transcoder{
		Source:          channel,
		CURRENT_QUALITY: nil,
		ON_REQ_TIMEOUT: func(t *engine.Transcoder) {
			fmt.Println("IPTV req timeout, destroying")
			t.Destroy("Request timeout not allowed on iptv")
		},
		ON_DESTROY: func(t *engine.Transcoder) {
			channel.Iptv.CurrentStreamCount -= 1
			index := -1
			for i, transcoder := range engine.Transcoders {
				if transcoder.UUID == t.UUID {
					index = i
					break
				}
			}
			if index != -1 {
				engine.Transcoders = append(engine.Transcoders[:index], engine.Transcoders[index+1:]...)
			}
			index = -1
			for i, transcoder := range channel.TranscodeIds {
				if transcoder == t.UUID {
					index = i
					break
				}
			}
			if index != -1 {
				channel.TranscodeIds = append(channel.TranscodeIds[:index], channel.TranscodeIds[index+1:]...)
			}
			index = -1
			for i, transcoder := range channel.Iptv.TranscodeIds {
				if transcoder == t.UUID {
					index = i
					break
				}
			}
			if index != -1 {
				channel.Iptv.TranscodeIds = append(channel.Iptv.TranscodeIds[:index], channel.Iptv.TranscodeIds[index+1:]...)
			}
		},
		UUID:              uuid.New().String(),
		OWNER_ID:          user.ID,
		ISLIVESTREAM:      true,
		FFURL:             dataStream,
		Task:              task,
		ON_PROGRESS:       func(i1, i2 int64) {},
		FFPROBE_TIMEOUT:   time.Second * 30,
		SEGMENTS:          make(map[string]chan func() io.Reader),
		Last_request_time: &ct,
		TRACKS:            make([]engine.AUDIO_TRACK, 0),
		SUBTITLES:         make([]engine.SUBTITLE, 0),
		CURRENT_TRACK:     -1,
		LENGTH:            -1,
		QUALITYS:          make([]engine.QUALITY, 0),
		CHUNK_LENGTH:      -1,
		CURRENT_INDEX:     -1,
		START_INDEX:       -1,
		FFMPEG:            nil,
		FETCHING_DATA:     false,
		Ffprobe:           nil,
		Request_pending:   false,
		Db:                db,
	}
	on_progress("transcoder", "Waiting ffpobe data")
	err = transcoder.GetData(stop)
	on_progress("transcoder", "Got ffprobe data")
	if err != nil {
		channel.Iptv.CurrentStreamCount -= 1
		transcoder.Destroy(err.Error())
		return err
	}
	channel.Iptv.TranscodeIds = append(channel.Iptv.TranscodeIds, transcoder.UUID)
	engine.Transcoders = append(engine.Transcoders, transcoder)
	channel.TranscodeIds = append(channel.TranscodeIds, transcoder.UUID)
	go transcoder.Start(0, transcoder.QUALITYS[0], 0)
	res := engine.TranscoderRes{
		Manifest:  engine.Config.Web.PublicUrl + "/transcode/" + transcoder.UUID + "/manifest",
		Uuid:      transcoder.UUID,
		Qualitys:  transcoder.QUALITYS,
		Tracks:    transcoder.TRACKS,
		Task_id:   task.ID,
		Subtitles: transcoder.SUBTITLES,
		Current:   0,
		Total:     0,
		Name:      channel.Name,
		Poster:    "https://image.tmdb.org/t/p/original/nHo5pZIIgAq2a9SJOUkfmldQHcB.jpg",
		Backdrop:  "https://image.tmdb.org/t/p/original/zYTUdAeUwHZLFqZXvMLhd1oOszh.jpg",
		Next:      engine.SKINNY_RENDER{},
		IsLive:    true,
	}
	if err != nil {
		return err
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	on_progress("transcoder", string(b))
	return nil
}
