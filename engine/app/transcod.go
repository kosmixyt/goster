package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"kosmix.fr/streaming/kosmixutil"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FormatZero = 5
)

var INTRO_DURATION = 4.02

type Transcoder struct {
	// watching must have user , movie | episode preloaded
	UUID            string
	Task            *Task
	FFPROBE_TIMEOUT time.Duration
	ISLIVESTREAM    bool
	// WATCHING        *WATCHING
	ON_PROGRESS    func(int64, int64)
	ON_REQ_TIMEOUT func(*Transcoder)
	SEGMENTS       map[string](chan (func() io.Reader))
	ON_DESTROY     func(*Transcoder)
	// on_progress func(int64, int64)
	CURRENT_QUALITY   *QUALITY
	QUALITYS          []QUALITY
	FFURL             interface{}
	LENGTH            float64
	CHUNK_LENGTH      int64
	TRACKS            []AUDIO_TRACK
	SUBTITLES         []SUBTITLE
	CURRENT_TRACK     int
	CURRENT_INDEX     int64
	START_INDEX       int64
	OWNER_ID          uint
	FETCHING_DATA     bool
	Ffprobe           *FFPROBE_DATA
	FFMPEG            *exec.Cmd
	Db                *gorm.DB
	Last_request_time *time.Time
	Request_pending   bool
	// *engine.IptvChannel || *engine.File
	Source interface{}
}

var Transcoders []*Transcoder

func (t *Transcoder) GetQuality(qualityName string) (QUALITY, error) {
	for _, quality := range t.QUALITYS {
		if quality.Name == qualityName {
			return quality, nil
		}
	}
	return QUALITY{}, fmt.Errorf("quality not found")
}

func (t *Transcoder) Manifest(Manifest chan string) {
	if !t.ISLIVESTREAM {
		if !t.FETCHING_DATA && t.LENGTH == -1 {
			panic("Must have data before manifest")
		}
		for {
			if t.LENGTH != -1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		manifest := []string{"#EXTM3U", "#EXT-X-VERSION:3", "#EXT-X-TARGETDURATION:" + strconv.Itoa(SEGMENT_TIME+1), "#EXT-X-MEDIA-SEQUENCE:0", "#EXT-X-PLAYLIST-TYPE:VOD"}
		duration := t.LENGTH
		for i := float64(0); i < float64(duration); i = i + SEGMENT_TIME {
			stime := SEGMENT_TIME
			if i+SEGMENT_TIME > float64(duration) {
				stime = float64(duration) - i
			}
			manifest = append(manifest, "#EXTINF:"+fmt.Sprintf("%.*f", 3, stime))
			index := i / SEGMENT_TIME
			manifest = append(manifest, Config.Web.PublicUrl+"/transcode/segment/"+t.UUID+"/"+strconv.Itoa(int(index)))
		}
		manifest = append(manifest, "#EXT-X-ENDLIST")
		Manifest <- strings.Join(manifest, "\n")
	} else {
		tm := time.Now()
		t.Last_request_time = &tm
		if t.FFMPEG == nil {
			t.Start(0, t.QUALITYS[0], 0)
		}
		manifest := []string{"#EXTM3U", "#EXT-X-TARGETDURATION:" + strconv.Itoa(SEGMENT_TIME+1), "#EXT-X-VERSION:3", "#EXT-X-MEDIA-SEQUENCE:0", "#EXT-X-PLAYLIST-TYPE:EVENT"}
		for i := 0; i < int(t.CURRENT_INDEX+1); i++ {
			manifest = append(manifest, "#EXTINF:"+fmt.Sprintf("%.*f", 3, SEGMENT_TIME))
			manifest = append(manifest, Config.Web.PublicUrl+"/transcode/segment/"+t.UUID+"/"+strconv.Itoa(i))
		}
		Manifest <- strings.Join(manifest, "\n")
	}
}

func (t *Transcoder) ManifestUrl() string {
	return Config.Web.PublicUrl + "/transcode/" + t.UUID + "/manifest"
}

func (t *Transcoder) GetData(stopChannel <-chan bool) error {
	t.FETCHING_DATA = true
	var data *FFPROBE_DATA
	var FloatDuration float64
out:
	for {
		select {
		case <-stopChannel:
			return fmt.Errorf("cancelled")
		default:
			t.Task.AddLog("Requesting Ffprobe Data")
			tempData, err := FfprobeData(t.FFURL, t.FFPROBE_TIMEOUT)
			if err != nil {
				fmt.Println("Error getting data", err)
				t.Task.AddLog("Error getting data", err.Error(), " Stopp")
				return err
			}
			if tempData.Format.Duration != "" {
				FloatDuration, err = strconv.ParseFloat(tempData.Format.Duration, 64)
				if err != nil {
					time.Sleep(100 * time.Millisecond)
					continue out
				}
			}
			if FloatDuration != 0 || t.ISLIVESTREAM {
				if len(tempData.Streams) > 0 {
					data = tempData
					t.Task.AddLog("Ffprobe Data fetched")
					break out
				}
			}
		}
	}
	for i, stream := range data.AudioStreams() {
		t.TRACKS = append(t.TRACKS, AUDIO_TRACK{
			Index: i,
			Name:  stream.Tags.Language,
		})
	}
	for i, stream := range data.SubtitleStreams() {
		t.SUBTITLES = append(t.SUBTITLES, SUBTITLE{
			Index: i,
			Name:  stream.Tags.Title + "(" + stream.Tags.Language + ")",
		})
	}
	t.QUALITYS = data.AdaptativeQualitys()
	t.CURRENT_QUALITY = &t.QUALITYS[0]
	t.LENGTH = FloatDuration
	t.Ffprobe = data
	t.FETCHING_DATA = false
	t.CURRENT_TRACK = data.FirstAudioStream().Index
	t.CURRENT_INDEX = 0
	t.CHUNK_LENGTH = int64(t.LENGTH / (SEGMENT_TIME))
	return nil
}
func (t *Transcoder) HasAudioStream(index int) bool {
	return 0 <= index && index <= len(t.TRACKS)
}
func (t *Transcoder) SetCurrentTime(time int64, currentIndex int) {
	// t.CURRENT_INDEX = int64(time)
	if time != 0 {
		t.ON_PROGRESS(time, int64(t.LENGTH)*int64(SEGMENT_TIME))
	} else {
		t.ON_PROGRESS(int64(currentIndex)*int64(SEGMENT_TIME), int64(t.LENGTH)*int64(SEGMENT_TIME))
	}
}
func (t *Transcoder) Chunk(index int, quality QUALITY, trackIndex int) (io.Reader, error) {
	t.Task.AddLog("Requesting chunk", strconv.Itoa(index))
	if index < 0 || (index > int(t.CHUNK_LENGTH) && !t.ISLIVESTREAM) {
		return nil, fmt.Errorf("invalid index")
	}
	if index > int(t.START_INDEX) && index < int(t.CURRENT_INDEX) && t.CURRENT_QUALITY.Name == quality.Name && t.CURRENT_TRACK == trackIndex {
		return ReadTranscodeFile(Joins(HLS_OUTPUT_PATH, (t.UUID)+"_"+t.CURRENT_QUALITY.Name+"_"+strconv.Itoa(t.CURRENT_TRACK)+"_"+fmt.Sprintf("%0"+strconv.Itoa(FormatZero)+"d", index)+".ts")), nil
	}
	if t.CURRENT_QUALITY.Name != quality.Name || t.CURRENT_TRACK != trackIndex {
		fmt.Println("bad quality")
		t.Start(index, quality, trackIndex)
	} else {
		if !t.ISLIVESTREAM {
			if int64(index) < t.START_INDEX {
				fmt.Println("to early")
				t.Start(index, quality, trackIndex)
			} else {
				if (t.CURRENT_INDEX + AVANCE) < int64(index) {
					fmt.Println("to late", t.CURRENT_INDEX, index)
					t.Start(index, quality, trackIndex)
				} else {
					if t.FFMPEG == nil {
						fmt.Println("nil ffmpeg")
						t.Start(index, quality, trackIndex)
					} else {
						if t.START_INDEX == -1 {
							fmt.Println("ffmpeg first run")
							t.Start(index, quality, trackIndex)
						} else {
							if t.CURRENT_TRACK != trackIndex {
								fmt.Println("track change")
								t.Start(index, quality, trackIndex)
							} else {
								fmt.Println("no need to start")
							}
						}
					}
				}
			}
		}
	}
	ii := fmt.Sprintf("%d_%s_%d", index, quality.Name, trackIndex)
	if t.SEGMENTS[ii] == nil {
		t.SEGMENTS[ii] = make(chan func() io.Reader)
	}
	fmt.Println("wait segment", index)
	select {
	case fnread := <-t.SEGMENTS[ii]:
		return fnread(), nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout")
	}
}

func (t *Transcoder) Destroy(why string) {
	if t.FFMPEG != nil {
		t.FFMPEG.Process.Kill()
	}
	t.Request_pending = false
	t.FFMPEG = nil
	t.ON_PROGRESS = nil
	t.Task.AddLog("Destroying transcoder" + why)
	t.Task.SetAsFinished()
	t.ON_DESTROY(t)
}
func (t *Transcoder) GetSubtitle(index int, result chan io.Reader) {
	if !t.FETCHING_DATA && t.LENGTH == -1 {
		go t.GetData(nil)
	}
	for {
		if t.LENGTH != -1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if index < 0 || index > len(t.SUBTITLES) {
		result <- nil
		return
	}
	args := []string{"-hide_banner", "-loglevel", "error"}
	if val, ok := t.FFURL.(string); ok {
		args = append(args, []string{
			"-i", val,
		}...)
	}
	if _, ok := t.FFURL.(io.ReadCloser); ok {
		args = append(args, []string{
			"-i", "pipe:0",
		}...)
	}
	args = append(args, []string{
		"-map", "0:s:" + strconv.Itoa(index),
		"-f", "webvtt",
		"pipe:1",
	}...)
	command := exec.Command(Config.Transcoder.FFMPEG, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		result <- nil
		return
	}
	if command.Start() != nil {
		result <- nil
		return
	}
	result <- stdout
	err = command.Wait()
	if err != nil {
		// panic(err)
	}
}
func (t *Transcoder) Start(index int, Quality QUALITY, trackIndex int) {
	// if t.Task == ni
	t.Task.AddLog("Starting ffmpeg", strconv.Itoa(index))
	if t.FFMPEG != nil {
		t.FFMPEG.Process.Kill()
		t.FFMPEG = nil
	}
	t.CURRENT_INDEX = int64(index)
	t.CURRENT_TRACK = trackIndex
	t.CURRENT_QUALITY = &Quality
	t.START_INDEX = int64(index)
	var args = []string{}
	if index > 0 {
		ss := (index * int(SEGMENT_TIME))
		args = append(args, "-ss", strconv.Itoa(ss))
	}
	cmdHead := []string{}
	fmt.Println("Start ffmpeg", t.FFURL)
	if val, ok := t.FFURL.(string); ok {
		cmdHead = append(cmdHead, []string{
			"-i", val,
		}...)
		if strings.HasPrefix(val, "http") {
			cmdHead = append(cmdHead, []string{
				"-tls_verify", "0",
				"-headers", "User-Agent: " + "curl/7.88.1" + "," + "Accept: */*" + "," + "Connection: keep-alive" + "," + "Accept-Encoding: gzip, deflate, br",
			}...)
		}
	}
	if _, ok := t.FFURL.(io.ReadCloser); ok {
		cmdHead = append(cmdHead, []string{
			"-i", "pipe:0",
		}...)
	}
	cmdVideo := []string{
		"-copyts",
	}
	cmdVideo = append(cmdVideo, kosmixutil.GetEncoderSettings("libx264")...)
	cmdVideo = append(cmdVideo,
		"-sc_threshold", "0",
		"-c:a", "libmp3lame",
		"-map_metadata", "-1",
		"-force_key_frames", "expr:gte(t,n_forced*"+strconv.FormatFloat(SEGMENT_TIME, 'f', -1, 64)+")",
		"-b:v", strconv.Itoa(Quality.VideoBitrate),
	)

	cmdVideo = append(cmdVideo, []string{
		"-threads", strconv.Itoa(Config.Transcoder.MaxTranscoderThreads),
		"-maxrate", "10000K",
		"-r", "30",
		"-map", "0:v:0",
		"-map", "0:a:" + strconv.Itoa(trackIndex),
		// to update
		"-s", strconv.Itoa(Quality.Width) + "x" + strconv.Itoa(Quality.Resolution),
		// "-s", "1280x720",
		"-b:a", strconv.Itoa(Quality.AudioBitrate) + "k",
	}...)
	cmdTail := []string{
		"-f", "segment",
		"-segment_time_delta", "0.1",
		"-segment_format", "mpegts",
		"-segment_list", "pipe:1",
		"-segment_time", strconv.FormatFloat(SEGMENT_TIME, 'f', -1, 64),
		"-segment_start_number", strconv.Itoa(index),
	}
	cmdFlags := []string{
		"-movflags", "+faststart",
		Joins(HLS_OUTPUT_PATH, (t.UUID)+"_"+Quality.Name+"_"+strconv.Itoa(trackIndex)+"_%0"+strconv.Itoa(FormatZero)+"d.ts"),
	}
	args = append(args, cmdHead...)
	args = append(args, cmdVideo...)
	args = append(args, cmdTail...)
	args = append(args, cmdFlags...)
	t.FFMPEG = exec.Command(Config.Transcoder.FFMPEG, args...)
	fmt.Println("Args", strings.Join(t.FFMPEG.Args, " "))
	stdout, err := t.FFMPEG.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := t.FFMPEG.StderrPipe()
	if err != nil {
		panic(err)
	}
	var stdin io.WriteCloser
	if _, ok := t.FFURL.(io.ReadCloser); ok {
		tempStdin, err := t.FFMPEG.StdinPipe()
		if err != nil {
			panic(err)
		}
		stdin = tempStdin
	}
	if err = t.FFMPEG.Start(); err != nil {
		panic(err)
	}
	if _, ok := t.FFURL.(io.ReadCloser); ok {
		if _, err = io.Copy(stdin, t.FFURL.(io.ReadCloser)); err != nil {
			panic(err)
		}
	}

	go func() {
		if t.FFMPEG != nil {
			err := t.FFMPEG.Wait()
			if err != nil {
				fmt.Println("ffmpeg error", err)
			}
		}
	}()
	go func() {
		currentInstance := t.FFMPEG.Process.Pid
		for {
			if t.FFMPEG == nil || t.FFMPEG.Process == nil {
				break
			}
			if currentInstance != t.FFMPEG.Process.Pid {
				break
			}
			if (*t.Last_request_time).Add(REQUEST_TIMEOUT).Before(time.Now()) && !t.Request_pending {
				fmt.Println("killing ffmpeg", REQUEST_TIMEOUT)
				t.ON_REQ_TIMEOUT(t)
				t.Task.AddLog("Request timeout killing ffmpeg")
				if t.FFMPEG != nil {
					t.FFMPEG.Process.Kill()
				}
				// fmt.Println("killed ffmpeg")
				t.Task.AddLog("Killed ffmpeg")
				t.FFMPEG = nil
				break
			}
			time.Sleep(40 * time.Millisecond)
			// fmt.Println(t.UUID, "waiting for ffmpeg to finish", t.Request_pending, (*t.Last_request_time).Add(REQUEST_TIMEOUT).Before(time.Now()))
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stdout)
		regex := regexp.MustCompile(`_([0-9]{5}).ts`)
		scanUUid := uuid.New().String()
		for scanner.Scan() {
			txt := scanner.Text()
			ProgressesIndex := regex.FindStringSubmatch(txt)
			if len(ProgressesIndex) < 2 {
				continue
			}
			intProgress, err := strconv.Atoi(ProgressesIndex[1])
			fmt.Println("stdout", scanner.Text(), "Progress Index", intProgress)
			t.Task.AddLog("Segment " + strconv.Itoa(intProgress) + " created")
			if err != nil {
				panic(err)
			}
			t.CURRENT_INDEX = int64(intProgress)
			go func() {
				ii := fmt.Sprintf("%d_%s_%d", intProgress, Quality.Name, trackIndex)
				if t.SEGMENTS[ii] == nil {
					t.SEGMENTS[ii] = make(chan func() io.Reader)
				}
				t.SEGMENTS[ii] <- func() io.Reader {
					return ReadTranscodeFile(Joins(HLS_OUTPUT_PATH, (t.UUID)+"_"+Quality.Name+"_"+strconv.Itoa(trackIndex)+"_"+ProgressesIndex[1]+".ts"))
				}
			}()
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(err, scanUUid)
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			// fmt.Println("stderr", text)
			t.Task.AddLog(text)
			// spl := strings.Split(scanner.Text(), "=")
			// if len(spl) != 2 {
			// 	panic("Error parsing ffmpeg output")
			// }
			// key, val := spl[0], spl[1]
			// if key == "speed" {
			// 	fmt.Println("speed", val)
			// }
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
	}()
	fmt.Println("End start")
}

type FFPROBE_DATA struct {
	Streams  []FFPROBE_STREAM  `json:"streams"`
	Format   FFPROBE_FORMAT    `json:"format"`
	Chapters []FFPROBE_CHAPTER `json:"chapters"`
}

func (f *FFPROBE_DATA) FirstVideoStream() FFPROBE_STREAM {
	for _, stream := range f.Streams {
		if stream.CodecType == "video" {
			return stream
		}
	}
	panic("No video stream")
}
func (f *FFPROBE_DATA) FirstAudioStream() FFPROBE_STREAM {
	for _, stream := range f.Streams {
		if stream.CodecType == "audio" {
			return stream
		}
	}
	panic("No audio stream")
}
func (f *FFPROBE_DATA) AdaptativeQualitys() []QUALITY {
	qualitys := make([]QUALITY, 5)
	bit_rate := f.Format.BitRate
	if bit_rate == "" {
		bit_rate = "10000000"
		fmt.Println("No bit rate generating for 1080p")
	}
	bitRate, err := strconv.Atoi(bit_rate)
	if err != nil {
		fmt.Println("Error parsing bit rate", bit_rate)
		return qualitys
	}
	var ofQual = make([]QUALITY, len(QUALITYS))
	copy(ofQual, QUALITYS)
	ofQual[4] = QUALITY{
		Name:         "240p",
		Resolution:   240,
		Width:        426,
		VideoBitrate: int(float64(bitRate) * 0.2),
		AudioBitrate: 64,
	}
	ofQual[3] = QUALITY{
		Name:         "360p",
		Resolution:   360,
		Width:        640,
		VideoBitrate: int(float64(bitRate) * 0.4),
		AudioBitrate: 96,
	}
	ofQual[2] = QUALITY{
		Name:         "480p",
		Resolution:   480,
		Width:        854,
		VideoBitrate: int(float64(bitRate) * 0.6),
		AudioBitrate: 128,
	}
	ofQual[1] = QUALITY{
		Name:         "720p",
		Resolution:   720,
		Width:        1280,
		VideoBitrate: int(float64(bitRate) * 0.8),
		AudioBitrate: 192,
	}
	ofQual[0] = QUALITY{
		Name:         "1080p",
		Resolution:   1080,
		Width:        1920,
		VideoBitrate: int(float64(bitRate) * 1),
		AudioBitrate: 256,
	}
	return ofQual
}
func (f *FFPROBE_DATA) AudioStreams() []FFPROBE_STREAM {
	streams := make([]FFPROBE_STREAM, 0)
	for _, stream := range f.Streams {
		if stream.CodecType == "audio" {
			streams = append(streams, stream)
		}
	}
	return streams
}
func (f *FFPROBE_DATA) AudioTrackByIndex(index int) *FFPROBE_STREAM {
	rindex := 0
	for _, stream := range f.Streams {
		if stream.CodecType == "audio" {
			if rindex == index {
				return &stream
			}
			rindex += 1
		}
	}
	return nil
	// panic("No audio stream")
}
func (f *FFPROBE_DATA) SubtitleStreams() []FFPROBE_STREAM {
	streams := make([]FFPROBE_STREAM, 0)
	currentSubIndex := 0
	for _, stream := range f.Streams {
		if stream.CodecType == "subtitle" {
			if stream.Codec != "dvd_subtitle" &&
				stream.Codec != "hdmv_pgs_subtitle" {
				stream.Index = currentSubIndex
				streams = append(streams, stream)
			}
			currentSubIndex += 1
		}
	}
	return streams

}
func (f *FFPROBE_DATA) VideoStreams() []FFPROBE_STREAM {
	streams := make([]FFPROBE_STREAM, 0)
	for _, stream := range f.Streams {
		if stream.CodecType == "video" {
			streams = append(streams, stream)
		}
	}
	return streams
}

// func GetStreamFromUrl(url string) io.ReadCloser {
// 	client := http.Client{
// 		Timeout: time.Second * 10,
// 		CheckRedirect: func(req *http.Request, via []*http.Request) error {
// 			return nil
// 		},
// 	}
// 	resp, err := client.Get(url)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return resp.Body
// }

type FFPROBE_STREAM struct {
	Index              int                        `json:"index"`
	Codec              string                     `json:"codec_name"`
	LongCodec          string                     `json:"codec_long_name"`
	Profile            string                     `json:"profile"`
	CodecType          string                     `json:"codec_type"`
	CodecTag           string                     `json:"codec_tag_string"`
	CodecTagString     string                     `json:"codec_tag"`
	Width              int                        `json:"width"`
	Height             int                        `json:"height"`
	Channels           int                        `json:"channels"`
	ChannelLayout      string                     `json:"channel_layout"`
	BitsPerSample      int                        `json:"bits_per_sample"`
	CodecWidth         int                        `json:"coded_width"`
	CodecHeight        int                        `json:"coded_height"`
	ClosedCaptions     int                        `json:"closed_captions"`
	FilmGrain          int                        `json:"film_grain"`
	HasBFrames         int                        `json:"has_b_frames"`
	SampleAspectRatio  string                     `json:"sample_aspect_ratio"`
	DisplayAspectRatio string                     `json:"display_aspect_ratio"`
	PixFmt             string                     `json:"pix_fmt"`
	Level              int                        `json:"level"`
	ColorRange         string                     `json:"color_range"`
	ChromaLocation     string                     `json:"chroma_location"`
	Refs               int                        `json:"refs"`
	RFrameRate         string                     `json:"r_frame_rate"`
	AvgFrameRate       string                     `json:"avg_frame_rate"`
	TimeBase           string                     `json:"time_base"`
	StartPts           int                        `json:"start_pts"`
	StartTime          string                     `json:"start_time"`
	ExtrataDataSize    int                        `json:"extradata_size"`
	Disposition        FFPROBE_STREAM_DISPOSITION `json:"disposition"`
	Tags               FFPROBE_TAG_STREAM         `json:"tags"`
}

type FFPROBE_TAG_STREAM struct {
	Language string `json:"language"`
	Title    string `json:"title"`
}

type FFPROBE_STREAM_DISPOSITION struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
	Captions        int `json:"captions"`
	Descriptions    int `json:"descriptions"`
	Metadata        int `json:"metadata"`
	Dependent       int `json:"dependent"`
	StillImage      int `json:"still_image"`
}
type FFPROBE_FORMAT struct {
	Filename       string       `json:"filename"`
	NbStreams      int          `json:"nb_streams"`
	NbPrograms     int          `json:"nb_programs"`
	FormatName     string       `json:"format_name"`
	FormatLongName string       `json:"format_long_name"`
	StartTime      string       `json:"start_time"`
	Duration       string       `json:"duration"`
	Size           string       `json:"size"`
	BitRate        string       `json:"bit_rate"`
	ProbeScore     int          `json:"probe_score"`
	Tags           FFPROBE_TAGS `json:"tags"`
}
type FFPROBE_TAGS struct {
	Title        string `json:"title"`
	Encoder      string `json:"encoder"`
	CreationTime string `json:"creation_time"`
}
type FFPROBE_CHAPTER struct {
}

func WriteFile(data io.Reader, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, data)
	return err
}
func Create206Allowed(app *gin.Engine, file *FILE, user *User) string {
	if user == nil || file == nil || app == nil {
		panic("Invalid params")
	}
	id := uuid.New().String()
	app.GET("/api/stream/"+id, func(ctx *gin.Context) {
		if reader := file.GetReader(); reader != nil {
			ctx.Writer.Header().Set("Content-Type", "video/mp4")
			ctx.Writer.Header().Set("Content-Disposition", "attachment; filename="+file.FILENAME)
			kosmixutil.ServerRangeRequest(ctx, file.SIZE, reader, false, false)
			return
		}
		if reader := file.GetNonSeekableReader(); reader != nil {
			kosmixutil.ServerNonSeekable(ctx, reader)
		}
	})
	return Config.Web.PublicUrl + "/stream/" + id
}

func GetFfUrl(app *gin.Engine, file *FILE) string {
	uniqid := uuid.New().String()
	app.GET("/api/temp/"+uniqid, func(ctx *gin.Context) {
		if reader := file.GetReader(); reader != nil {
			ctx.Writer.Header().Set("Content-Disposition", "attachment; filename="+file.FILENAME)
			kosmixutil.ServerRangeRequest(ctx, file.SIZE, reader, true, true)
			return
		}
		if reader := file.GetNonSeekableReader(); reader != nil {
			kosmixutil.ServerNonSeekable(ctx, reader)
		}
	})
	return GetPrivateUrl() + "/api/temp/" + uniqid
}

// func GetFinalURL(url string, userAgent string) (string, error) {
// 	fmt.Println("GetFinalURL", url)
// 	client := &http.Client{
// 		CheckRedirect: func(req *http.Request, via []*http.Request) error {
// 			// Empêche le client de suivre automatiquement les redirections
// 			return http.ErrUseLastResponse
// 		},
// 	}

// 	// Création de la requête HEAD avec un User-Agent personnalisé
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Set("User-Agent", userAgent)
// 	req.Header.Set("Accept", "*/*")
// 	req.Header.Set("Connection", "keep-alive")
// 	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

// 	// Exécution de la requête
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	resp.Body.Close()

// 	fmt.Println(resp.StatusCode)
// 	fmt.Println(resp.Header)
// 	// Vérifier si le statut indique une redirection (codes 3xx)
// 	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
// 		location := resp.Header.Get("Location")
// 		if location != "" {
// 			return location, nil
// 		}
// 	}

//		// Retourner l'URL originale si aucune redirection
//		return url, nil
//	}
func FfprobeData(source interface{}, timeout time.Duration) (*FFPROBE_DATA, error) {
	args := []string{}
	// if val, ok := source.(string); ok {
	// 	// if strings.HasPrefix(val, "http") {
	// 	// 	args = append(args, []string{
	// 	// 		"-headers", "User-Agent: " + "curl/7.88.1" + "," + "Accept: */*" + "," + "Connection: keep-alive" + "," + "Accept-Encoding: gzip, deflate, br",
	// 	// 	}...)
	// 	// }
	// }
	args = append(args, []string{
		"-show_format",
		"-show_streams",
		"-show_private_data",
		"-print_format", "json",
	}...)
	if val, ok := source.(string); ok {
		args = append(args, []string{
			val,
		}...)
	}
	if _, ok := source.(io.ReadCloser); ok {
		args = append(args, []string{
			"pipe:0",
		}...)
	}

	command := exec.Command(Config.Transcoder.FFPROBE, args...)
	command.Env = os.Environ()

	fmt.Println(strings.Join(command.Args, " "))
	var out bytes.Buffer
	var stde bytes.Buffer
	command.Stdout = &out
	command.Stderr = &stde
	var stdin io.WriteCloser
	if _, ok := source.(io.ReadCloser); ok {
		tempStdin, err := command.StdinPipe()
		if err != nil {
			return &FFPROBE_DATA{}, err
		}
		stdin = tempStdin
	}
	if err := command.Start(); err != nil {
		return &FFPROBE_DATA{}, err
	}
	if _, ok := source.(io.ReadCloser); ok {
		if _, err := io.Copy(stdin, source.(io.ReadCloser)); err != nil {
			return &FFPROBE_DATA{}, err
		}
	}

	err := command.Wait()
	if err != nil {
		fmt.Println(stde.String())
		return &FFPROBE_DATA{}, err
	}
	var ffprobeData FFPROBE_DATA
	if err := json.Unmarshal(out.Bytes(), &ffprobeData); err != nil {
		return &ffprobeData, err
	}
	fmt.Println(ffprobeData)
	return &ffprobeData, nil
}

func ReadTranscodeFile(path string) io.Reader {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	return file
}

func captureScreenshot(videoPath, outputImage string, targetTime float64) error {
	timeStr := fmt.Sprintf("%.2f", targetTime) // Formater le temps en chaîne
	cmd := exec.Command("ffmpeg", "-ss", timeStr, "-i", videoPath, "-frames:v", "1", "-q:v", "2", outputImage)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
