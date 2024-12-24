package engine

import (
	"bufio"
	"errors"
	"time"

	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kosmix.fr/streaming/kosmixutil"
)

var Converts []*Convert

type Convert struct {
	Start            time.Time
	SOURCE_FILE      *FILE
	OUTPUT_FILE      *FILE
	OUTPUT_FILE_NAME string
	User             *User
	AudioTrackIndex  uint
	Paused           bool
	Command          *exec.Cmd
	Task             *Task
	Quality          *QUALITY
	Progress         *FfmpegProgress
	FfprobeBase      *FFPROBE_DATA
	Running          bool
	OutputPathStorer *StoragePathElement
}

func GetConvertByTaskId(task_id uint) *Convert {
	for _, c := range Converts {
		if c.Task.ID == task_id {
			return c
		}
	}
	return nil
}
func (c *Convert) Pause() error {
	if c.Command == nil {
		return errors.New("command is nil")
	}
	if err := kosmixutil.PauseExec(c.Command); err != nil {
		return err
	}
	c.Paused = true
	return nil
}
func (c *Convert) Resume() error {
	if c.Command == nil {
		return errors.New("command is nil")
	}
	if err := kosmixutil.ResumeExec(c.Command); err != nil {
		return err
	}
	c.Paused = false
	return nil
}
func (c *Convert) Stop() error {
	if c.Command == nil {
		return errors.New("command is nil")
	}
	c.Command.Process.Kill()
	c.Paused = false
	c.Task.SetAsError(errors.New("stopped"))
	return nil
}

func (c *Convert) Convert(app *gin.Engine) (*FILE, error) {
	if c.OUTPUT_FILE != nil || c.SOURCE_FILE == nil || c.User == nil {
		return nil, errors.New("invalid convert")
	}
	if !c.User.CAN_CONVERT {
		return nil, c.Task.SetAsError(errors.New("user can't convert")).(error)
	}
	if c.Quality == nil {
		return nil, c.Task.SetAsError(errors.New("quality not found")).(error)
	}
	if c.FfprobeBase.AudioTrackByIndex(int(c.AudioTrackIndex)) == nil {
		return nil, c.Task.SetAsError(errors.New("invalid audio_track_index " + strconv.Itoa(len(c.FfprobeBase.AudioStreams())))).(error)
	}
	c.Task.SetAsStarted()
	output_filename := "convert-" + strconv.Itoa(c.Quality.Width) + "-" + ReplaceExtension(c.SOURCE_FILE.FILENAME, ".mp4")
	c.OUTPUT_FILE_NAME = output_filename
	c.Task.AddLog("Converting " + c.SOURCE_FILE.FILENAME + " to " + strconv.Itoa(c.Quality.Width) + "p with new path" + c.OutputPathStorer.Path)
	args := []string{
		"-i",
		"pipe:0",
		"-sn",
	}
	args = append(args, []string{
		"-map",
		"0:v:0",
		"-map",
		"0:a:" + strconv.Itoa(int(c.AudioTrackIndex)),
	}...)
	args = append(args, []string{
		"-threads",
		strconv.Itoa(Config.Transcoder.MaxConverterThreads),
		"-b:v", strconv.Itoa(c.Quality.VideoBitrate) + "k",
	}...)
	args = append(args, kosmixutil.GetEncoderSettings("libx264")...)
	args = append(args, []string{
		"-c:a",
		"libmp3lame",
		"-b:a",
		strconv.Itoa(c.Quality.AudioBitrate) + "k",
	}...)
	ffmpeg_output, on_finish, err := CreateTempFfmpegOutputFile(c.OutputPathStorer, c.OUTPUT_FILE_NAME)
	if err != nil {
		c.Task.SetAsError(err)
		return nil, err
	}
	args = append(args, []string{
		"-progress",
		"pipe:2",
		"-f",
		"mp4",
		ffmpeg_output,
		"-y"}...)
	fmt.Println("ffmpeg", args)
	c.Command = exec.Command(Config.Transcoder.FFMPEG, args...)
	read := c.SOURCE_FILE.GetReader()
	if read == nil {
		(*on_finish)(true, c.Task)
		panic("read is nil")
	}
	stdin, err := c.Command.StdinPipe()
	if err != nil {
		(*on_finish)(true, c.Task)
		panic(err)
	}
	Converts = append(Converts, c)
	stderr, err := c.Command.StderrPipe()
	if err != nil {
		(*on_finish)(true, c.Task)
		panic(err)
	}
	go func() {
		io.Copy(stdin, read)
		stdin.Close()
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		durationAsInt, err := strconv.ParseFloat(c.FfprobeBase.Format.Duration, 64)
		if err != nil {
			fmt.Println("Error parsing duration", err)
			durationAsInt = 1
		}
		for scanner.Scan() {
			text := scanner.Text()
			ParseFfmpegOutput(text, int64(durationAsInt), c.Progress)
			c.Task.AddLog(text)
		}
		fmt.Println("END SCAN")
	}()
	err = c.Command.Start()
	c.Running = true
	if err != nil {
		c.Running = false
		(*on_finish)(true, c.Task)
		return nil, c.Task.SetAsError(err).(error)
	}
	fmt.Println("Convert started")
	err = c.Command.Wait()
	c.Running = false
	c.Command = nil
	if err != nil {
		(*on_finish)(true, c.Task)
		return nil, c.Task.SetAsError(err).(error)
	}
	fmt.Println("Convert finished")
	(*on_finish)(false, c.Task)
	Converts = append(Converts, c)
	output_file := &FILE{
		MOVIE_ID:           c.SOURCE_FILE.MOVIE_ID,
		EPISODE_ID:         c.SOURCE_FILE.EPISODE_ID,
		SEASON_ID:          c.SOURCE_FILE.SEASON_ID,
		TV_ID:              c.SOURCE_FILE.TV_ID,
		IS_MEDIA:           true,
		FILENAME:           output_filename,
		StoragePathElement: c.OutputPathStorer,
		SUB_PATH:           "",
		SIZE:               0,
	}
	stats, err := output_file.stats()
	if err != nil {
		panic(err)
	}
	output_file.SIZE = stats.Size()
	c.OUTPUT_FILE = output_file
	db.Create(output_file)
	DeleteFromConverts(c)
	c.Task.SetAsFinished()
	return output_file, nil
}

func DeleteFromConverts(c *Convert) {
	for i, conv := range Converts {
		if conv == c {
			Converts = append(Converts[:i], Converts[i+1:]...)
			return
		}
	}
}

func ReplaceExtension(filename string, newExtension string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i] + newExtension
		}
	}
	return filename
}

// frame=24237 fps=155 q=33.0 size=  134144kB time=00:16:09.93 bitrate=1133.0kbits/s dup=2 drop=0 speed=6.21x
// frame=24237
// fps=155.16
// stream_0_0_q=33.0
// bitrate=1133.0kbits/s
// total_size=137363504
// out_time_us=969936479
// out_time_ms=969936479
// out_time=00:16:09.936479
// dup_frames=2
// drop_frames=0
// speed=6.21x
// progress=continue

type FfmpegProgress struct {
	Frame         int
	Fps           float64
	Stream_0_0_q  float64
	Bitrate       float64
	Total_size    int64
	Out_time_us   int64
	Out_time_ms   int64
	Out_time      string
	Dup_frames    int
	Drop_frames   int
	Speed         float64
	Progress      string
	TotalProgress float64
}

func ParseFfmpegOutput(text string, base_duration int64, progress *FfmpegProgress) {
	splitted := strings.Split(text, " ")
	result := make(map[string]string)
	for _, s := range splitted {
		splitted := strings.Split(s, "=")
		if len(splitted) != 2 {
			continue
		}
		result[splitted[0]] = splitted[1]
	}
	if result["frame"] != "" {
		progress.Frame, _ = strconv.Atoi(result["frame"])
	}

	if result["fps"] != "" {
		progress.Fps, _ = strconv.ParseFloat(result["fps"], 64)
	}
	if result["stream_0_0_q"] != "" {
		progress.Stream_0_0_q, _ = strconv.ParseFloat(result["stream_0_0_q"], 64)
	}
	if result["bitrate"] != "" {
		if len(result["bitrate"]) > 6 {
			progress.Bitrate, _ = strconv.ParseFloat(result["bitrate"][:len(result["bitrate"])-6], 64)
		}
	}
	if result["total_size"] != "" {
		progress.Total_size, _ = strconv.ParseInt(result["total_size"], 10, 64)
	}
	if result["out_time_us"] != "" {
		progress.Out_time_us, _ = strconv.ParseInt(result["out_time_us"], 10, 64)
	}
	if result["out_time_ms"] != "" {
		progress.Out_time_ms, _ = strconv.ParseInt(result["out_time_ms"], 10, 64)
	}
	if result["out_time"] != "" {
		progress.Out_time = result["out_time"]
	}
	if result["dup_frames"] != "" {
		progress.Dup_frames, _ = strconv.Atoi(result["dup_frames"])
	}
	if result["drop_frames"] != "" {
		progress.Drop_frames, _ = strconv.Atoi(result["drop_frames"])
	}
	if result["speed"] != "" {
		progress.Speed, _ = strconv.ParseFloat(result["speed"][:len(result["speed"])-1], 64)
	}
	if result["progress"] != "" {
		progress.Progress = result["progress"]
		progress.TotalProgress = 1
	}
	progress.TotalProgress = ((float64(progress.Out_time_us) / float64(1000_000)) / float64(base_duration))
}
