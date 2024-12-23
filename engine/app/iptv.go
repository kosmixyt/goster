package engine

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

var Iptvs []*IptvItem = make([]*IptvItem, 0)
var reg = regexp.MustCompile(`#EXTINF:-1\s*tvg-id="([^"]*)"\s*tvg-name="([^"]*)"\s*tvg-logo="([^"]*)"\s*group-title="([^"]*)",[^\n]*\n([^\n]*)`)

func ReadFile(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return body
}
func (channel *IptvChannel) Skinny() SKINNY_RENDER {
	return SKINNY_RENDER{
		ID:            strconv.FormatInt(channel.Id, 10),
		WATCH:         WatchData{},
		TYPE:          "iptv",
		NAME:          channel.Name,
		LOGO:          channel.Logo_url,
		BACKDROP:      channel.Logo_url,
		POSTER:        channel.Logo_url,
		TRAILER:       "",
		DESCRIPTION:   fmt.Sprintf("Iptv channel %s", channel.Name),
		YEAR:          0,
		RUNTIME:       "",
		GENRE:         []GenreItem{},
		WATCHLISTED:   false,
		TRANSCODE_URL: "",
		PROVIDERS:     []PROVIDERItem{},
		DisplayData:   "",
	}
}
func (iptv *IptvItem) GetGroup(name string) *IptvGroup {
	if name == "" {
		return nil
	}
	for _, group := range iptv.Groups {
		if group.Name == name {
			return group
		}
	}
	return nil
}
func AppendIptv(iptv *IptvItem) {
	Iptvs = append(Iptvs, iptv)
}
func (iptv *IptvItem) AddGroup(name string) *IptvGroup {
	group := iptv.GetGroup(name)
	if group == nil {
		group = &IptvGroup{
			Name:     name,
			Channels: make([]*IptvChannel, 0),
		}
		iptv.Groups = append(iptv.Groups, group)
	}
	return group
}
func (iptv *IptvItem) AddChannel(channel *IptvChannel) {
	iptv.Channels = append(iptv.Channels, channel)
}

var Fid int64 = 0

func InitIptv(db *gorm.DB) bool {
	var preloadIptv []IptvItem
	db.
		Preload("RECORDS").
		Find(&preloadIptv)
	for _, iptv := range preloadIptv {
		iptv.Init(&Fid)
		AppendIptv(&iptv)
	}
	return true
}
func (iptv *IptvItem) Init(offset *int64) {
	iptv.TranscodeIds = make([]string, 0)
	iptv.CurrentStreamCount = 0
	iptv.Channels = make([]*IptvChannel, 0)
	iptv.Groups = make([]*IptvGroup, 0)
	strbody := string(ReadFile(Joins(IPTV_M3U8_PATH, iptv.FileName)))
	matches := reg.FindAllStringSubmatch(strbody, -1)
	for _, match := range matches {
		*offset += 1
		group := match[4]
		url := match[5]
		tvlogo := match[3]
		name := match[2]
		channel := &IptvChannel{
			TranscodeIds: make([]string, 0),
			Id:           *offset,
			Name:         name,
			Logo_url:     tvlogo,
			Url:          url,
			Group:        nil,
			Iptv:         iptv,
		}
		if group != "" {
			gp := iptv.GetGroup(group)
			if gp == nil {
				iptv.AddGroup(group).Channels = append(iptv.GetGroup(group).Channels, channel)
			} else {
				gp.Channels = append(gp.Channels, channel)
			}
		}
		iptv.AddChannel(channel)
	}
}
func TestTextIptv(text string) error {
	matches := reg.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return fmt.Errorf("no match found")
	}
	for _, match := range matches {
		fmt.Println(len(match))
		if len(match) < 5 {
			return fmt.Errorf("invalid match %v", match)
		}
	}
	return nil
}

func (iptv *IptvItem) ListIptv(offset int64, limit int64, group *string) []*IptvChannel {
	if group != nil {
		gp := iptv.GetGroup(*group)
		if gp != nil {
			if offset+limit > int64(len(gp.Channels)) {
				if offset > int64(len(gp.Channels)) {
					return make([]*IptvChannel, 0)
				}
				limit = int64(len(gp.Channels)) - offset
			}
			return gp.Channels[offset : offset+limit]
		}
		return make([]*IptvChannel, 0)
	}
	if offset+limit > int64(len(iptv.Channels)) {
		if offset > int64(len(iptv.Channels)) {
			return make([]*IptvChannel, 0)
		}
		limit = int64(len(iptv.Channels)) - offset
	}
	return iptv.Channels[offset : offset+limit]
}
func MapIptvToRender(iptv []*IptvChannel) []*IptvChannelRender {
	if len(iptv) == 0 {
		return make([]*IptvChannelRender, 0)
	}
	item := iptv[0].Iptv
	items := make([]*IptvChannelRender, 0)
	for _, channel := range iptv {
		var gn string
		if channel.Group != nil {
			gn = channel.Group.Name
		}
		items = append(items, &IptvChannelRender{
			Id:            channel.Id,
			Name:          channel.Name,
			Logo_url:      Config.Web.PublicUrl + "/iptv/logo?id=" + strconv.Itoa(int(item.ID)) + "&channel=" + strconv.Itoa(int(channel.Id)),
			GroupName:     gn,
			TRANSCODE_URL: Config.Web.PublicUrl + "/iptv/transcode?channel=" + strconv.Itoa(int(channel.Id)),
		})
	}
	return items
}

func (iptv *IptvItem) GetChannel(channel_id int) *IptvChannel {
	for _, channel := range iptv.Channels {
		if channel.Id == int64(channel_id) {
			return channel
		}
	}
	return nil
}

func (ptv *IptvItem) public() bool {
	return strings.Contains(ptv.FileName, "public")
}

func (record *Record) LoadTask() {
	if record.TASK_ID == 0 {
		panic("Task id not found")
	}
	db.Model(&record).Association("Task").Find(&record.Task)
	if record.Task == nil {
		panic("Task not found")
	}
	Tasks = append(Tasks, record.Task)
}
func (record *Record) LoadOwner() {
	if record.OWNER_ID == 0 {
		panic("Owner not found")
	}
	db.Model(&record).Association("OWNER").Find(&record.OWNER)
	if record.OWNER == nil {
		panic("Owner not found")
	}
}

func (record *Record) Init() {
	record.LoadTask()
	record.LoadOwner()
	record.Task.AddLog("Waiting for record to start", strconv.FormatFloat(time.Until(record.START).Seconds(), 'f', -1, 64))
	until := time.Until(record.START)
	if until.Seconds() < 0 {
		db.Updates(Record{ERROR: "Record already started"}).Model(&record)
		return
	}
	fmt.Println("Sleeping for", until.Seconds(), "seconds")
	time.Sleep(until)
	fmt.Println("Record started")
	iptv := record.OWNER.GetIptvById(int(record.IPTV_ID))
	if iptv == nil {
		db.Model(&record).Updates(Record{ERROR: "Iptv not found"})
		return
	}
	channel := iptv.GetChannel(int(record.CHANNEL_ID))
	if channel == nil {
		db.Model(&record).Updates(Record{ERROR: "Channel not found", ENDED: true})
		return
	}
	ffmpeg_output, on_finish, err := CreateTempFfmpegOutputFile(record.OutputPathStorer, record.OutputStorerFileName)
	if err != nil {
		db.Model(&record).Updates(Record{ERROR: err.Error(), ENDED: true})
		return
	}
	fmt.Println("Output file", ffmpeg_output)
	if iptv.CurrentStreamCount >= iptv.MaxStreamCount {
		if !record.Force {
			fmt.Println("Max stream reached cannot record")
			(*on_finish)(true, record.Task)
			db.Model(&record).Updates(Record{ERROR: "Max stream reached cannot record", ENDED: true})
			return
		} else {
			user_transcoders := record.OWNER.GetUserTranscoders()
			get_this_iptv := iptv.TranscodeIds
			intersection := make([]*Transcoder, 0)
			for _, transcoder := range user_transcoders {
				for _, transcode_uid := range get_this_iptv {
					if transcoder.UUID == transcode_uid {
						intersection = append(intersection, transcoder)
					}
				}
			}
			if len(intersection) == 0 {
				(*on_finish)(true, record.Task)
				db.Model(&record).Updates(Record{
					ERROR: "No transcoder available",
					ENDED: true,
				})
				return
			}
			victime := intersection[0]
			victime.Destroy("Init record force" + strconv.Itoa(int(record.ID)))
		}
	}
	if err != nil {
		(*on_finish)(true, record.Task)
		db.Model(&record).Updates(Record{
			ERROR: err.Error(),
			ENDED: true,
		})
	}
	iptv.CurrentStreamCount += 1
	args := append([]string{
		"-i", channel.Url[0 : len(channel.Url)-1],
		"-tls_verify", "0",
		"-headers", "User-Agent: curl/7.88.1,Accept: */*,Connection: keep-alive,Accept-Encoding: gzip, deflate, br",
		"-t", strconv.Itoa(int(record.DURATION)),
	}, kosmixutil.GetEncoderSettings("libx264")...)
	args = append(args,
		"-c:a", "libmp3lame",
		"-s", "1280x720",
		"-f", "mp4",
		ffmpeg_output,
		"-y",
	)
	cmd := exec.Command(Config.Transcoder.FFMPEG, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		(*on_finish)(true, record.Task)
		panic(err)
	}
	go io.Copy(os.Stdout, stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		(*on_finish)(true, record.Task)
		panic(err)

	}
	fmt.Println("Recording started")
	cmd.Start()
	go io.Copy(os.Stderr, stderr)
	err = cmd.Wait()
	iptv.CurrentStreamCount -= 1
	if err != nil {
		(*on_finish)(true, record.Task)
		db.Model(&record).Updates(Record{
			ERROR: err.Error(),
			ENDED: true,
		})
		return
	}

	if err := (*on_finish)(false, record.Task); err != nil {
		db.Model(&record).Updates(Record{
			ERROR: err.Error(),
			ENDED: true,
		})
		return
	}
	stats, err := record.OutputStorerMem.Conn.Stats(Joins(record.OutputPathStorer.Path, record.OutputStorerFileName))
	if err != nil {
		db.Model(&record).Updates(Record{
			ERROR: err.Error(),
			ENDED: true,
		})
		return
	}
	newFile := &FILE{
		StoragePathElement: record.OutputPathStorer,
		SourceRecord:       record,
		SourceRecordID:     &record.ID,
		SUB_PATH:           "",
		FILENAME:           record.OutputStorerFileName,
		IS_MEDIA:           true,
		SIZE:               stats.Size(),
	}
	if record.OUTPUT_EPISODE != nil {
		newFile.EPISODE_ID = *record.OUTPUT_EPISODE_ID
		var episode *EPISODE
		db.Preload("SEASON").Where("id = ?", *record.OUTPUT_EPISODE_ID).First(&episode)
		if episode == nil {
			panic("Episode not found but it should be")
		}
		newFile.TV_ID = episode.SEASON.TV_ID
		newFile.SEASON_ID = episode.SEASON_ID
	}
	if record.OUTPUT_MOVIE_ID != nil {
		newFile.MOVIE_ID = *record.OUTPUT_MOVIE_ID
	}
	db.Save(&newFile)
	db.Model(record).Update("OUTPUT_FILE_ID", newFile.ID)
	db.Model(&record).Updates(Record{ENDED: true})
}
func InitRecords(db *gorm.DB) {
	var records []*Record
	db.Preload("OWNER").Find(&records)
	for _, record := range records {
		if !record.START.Before(time.Now()) && !record.ENDED {
			go record.Init()
		} else {
			fmt.Println("Record already ended", record.ENDED, record.START.Before(time.Now()))
		}
	}
}

func GetIptvFileFromUrl(url string) (io.Reader, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "ExoPlayer v2.12.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
func CreateTempFfmpegOutputFile(outputStorer *StoragePathElement, fileName string) (string, *func(bool, *Task) error, error) {
	nilfunc := func(f bool, task *Task) error {
		return nil
	}
	// if !outputStorer.getStorage() {
	// return Joins(output_root_path, fileName), &nilfunc, nil
	// }
	if outputStorer.getStorage() == nil {
		return Joins(Config.Torrents.DownloadPath, fileName), &nilfunc, nil
	}
	if !outputStorer.getStorage().toConn().NeedProxy() {
		return Joins(outputStorer.Path, fileName), &nilfunc, nil
	}
	file_name_extension := filepath.Ext(fileName)
	temp_file_name := uuid.NewString() + file_name_extension
	temp_file_path := Joins(FFMPEG_BIG_FILE_PATH, temp_file_name)
	called := false
	on_finish := func(cancel bool, task *Task) error {
		if called {
			panic("on_finish called twice")
		}
		called = true
		if cancel {
			return os.Remove(temp_file_path)
		}
		writer, err := outputStorer.getStorage().toConn().GetWriter(Joins(outputStorer.Path, fileName))
		if err != nil {
			return err
		}
		reader, err := os.Open(temp_file_path)
		if err != nil {
			panic(err)
		}
		task.AddLog("Copying file to storage", "")
		_, err = io.Copy(writer, reader)
		task.AddLog("File copied to storage", "")
		if err != nil {
			panic(err)
		}
		err = os.Remove(temp_file_path)
		if err != nil {
			panic(err)
		}
		return nil
	}
	return temp_file_path, &on_finish, nil

}
