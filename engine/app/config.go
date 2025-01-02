package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

const ServerSideError = "serverError"
const DOWNLOAD_LOCAL_TRAILER = true
const Movie = "movie"
const Tv = "tv"
const TMDB_HIGH = "https://image.tmdb.org/t/p/w780/"
const TMDB_LOW string = "https://image.tmdb.org/t/p/w500/"

var TMDB_ORIGINAL string = "https://image.tmdb.org/t/p/original/"

var IMG_PATH string
var IPTV_M3U8_PATH string
var FILES_TORRENT_PATH string
var HLS_OUTPUT_PATH string
var TRAILER_OUTPUT_PATH string
var FFMPEG_BIG_FILE_PATH string

var Config = AppConfig{}
var NewConfig = AppConfig{}

func SetupCachePaths(cache_path string) {
	abs, err := filepath.Abs(cache_path)
	if err != nil {
		panic(err)
	}
	IMG_PATH = Joins(abs, "imgs")
	IPTV_M3U8_PATH = Joins(abs, "iptv")
	FILES_TORRENT_PATH = Joins(abs, "torrents")
	HLS_OUTPUT_PATH = Joins(abs, "hls")
	TRAILER_OUTPUT_PATH = Joins(abs, "trailer")
	FFMPEG_BIG_FILE_PATH = Joins(abs, "ffmpeg")
	PanicOnError(os.MkdirAll(cache_path, os.ModePerm))
	PanicOnError(os.MkdirAll(IMG_PATH, os.ModePerm))
	PanicOnError(os.MkdirAll(IPTV_M3U8_PATH, os.ModePerm))
	PanicOnError(os.MkdirAll(FILES_TORRENT_PATH, os.ModePerm))
	PanicOnError(os.MkdirAll(HLS_OUTPUT_PATH, os.ModePerm))
	PanicOnError(os.MkdirAll(TRAILER_OUTPUT_PATH, os.ModePerm))
	PanicOnError(os.MkdirAll(FFMPEG_BIG_FILE_PATH, os.ModePerm))
}
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

type StorageElement struct {
	TYPE    string                   `json:"type"`
	Options interface{}              `json:"options"`
	Paths   []kosmixutil.PathElement `json:"paths"`
	Name    string                   `json:"name"`
}

// const FFPROBE_TIMEOUT = 10 * time.Second
// const SEGMENT_TIME = 2.0

const AVANCE = 1

type AppConfig struct {
	Locations []StorageElement `json:"scan_paths"`
	DB        struct {
		Driver   string `json:"driver"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"db"`
	Cert struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
	} `json:"cert"`
	Limits struct {
		MovieSize     int64 `json:"movie_size"`
		SeasonSize    int64 `json:"season_size"`
		CheckInterval int   `json:"check_interval"`
	} `json:"limits"`
	Transcoder struct {
		EnableForWebPlayableFiles bool      `json:"enable_for_web_playable_files"`
		MaxTranscoderThreads      int       `json:"max_transcoder_threads"`
		MaxConverterThreads       int       `json:"max_converter_threads"`
		FFMPEG                    string    `json:"ffmpeg"`
		FFPROBE                   string    `json:"ffprobe"`
		FFPROBE_TIMEOUT           uint      `json:"ffprobe_timeout"`
		SEGMENT_TIME              float64   `json:"segment_time"`
		REQUEST_TIMEOUT           uint      `json:"request_timeout"`
		Qualitys                  []QUALITY `json:"qualitys"`
	} `json:"transcoder"`
	Torrents struct {
		DownloadPath string `json:"download_path"`
	} `json:"torrents"`
	Web struct {
		PublicPort  string `json:"public_port"`
		PublicUrl   string `json:"public_url"`
		CrossOrigin string `json:"cross_origin"`
	} `json:"web"`
	Metadata struct {
		Tmdb                    string         `json:"tmdb"`
		TmdbIso3166             string         `json:"tmdb_iso3166"`
		TmdbMovieWatchProviders map[string]int `json:"tmdb_movie_watch_providers"`
		TmdbTvWatchProviders    map[string]int `json:"tmdb_tv_watch_providers"`
		Omdb                    string         `json:"omdb"`
		TmdbLang                string         `json:"tmdb_lang"`
		TmdbImgLang             []string       `json:"tmdb_lang_imgs"`
	} `json:"metadata"`
	CachePath  string `json:"cache_path"`
	Cloudflare struct {
		ChallengeResolver  string `json:"challenge_resolver"`
		FlaresolverrUrl    string `json:"flaresolverr_url"`
		CapsolverrProxyUrl string `json:"capsolverr_proxy_url"`
		CapsolverrApiKey   string `json:"capsolverr_api_key"`
	} `json:"cloudflare"`
	TorrentProviders map[string]map[string]string `json:"torrent_providers"`
}

// TorrentProviders struct {
// 	Sharewood map[string]string `json:"sharewood"`
// 	YGG       map[string]string `json:"ygg"`
// } `json:"torrent_providers"`
// TorrentProviders struct {
// 	Sharewood struct {
// 		Key      string `json:"key"`
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	} `json:"sharewood"`
// 	YGG struct {
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	} `json:"ygg"`

func LoadConfig() {
	f, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&Config); err != nil {
		panic(err)
	}
	NewConfig = Config
	kosmixutil.InitKeys(Config.Metadata.Tmdb, Config.Metadata.Omdb, Config.Metadata.TmdbImgLang, Config.Metadata.TmdbLang)
	SetupCachePaths(Config.CachePath)

}

type TranscoderEditableSettings struct {
	EnableForWebPlayableFiles bool    `json:"enable_for_web_playable_files"`
	MaxTranscoderThreads      int     `json:"max_transcoder_threads"`
	MaxConverterThreads       int     `json:"max_converter_threads"`
	FFMPEG                    string  `json:"ffmpeg"`
	FFPROBE                   string  `json:"ffprobe"`
	FFPROBE_TIMEOUT           uint    `json:"ffprobe_timeout"`
	SEGMENT_TIME              float64 `json:"segment_time"`
	REQUEST_TIMEOUT           uint    `json:"request_timeout"`
}

func (f *TranscoderEditableSettings) VerifyAndSet() error {
	Config.Transcoder.EnableForWebPlayableFiles = f.EnableForWebPlayableFiles
	Config.Transcoder.MaxTranscoderThreads = f.MaxTranscoderThreads
	Config.Transcoder.MaxConverterThreads = f.MaxConverterThreads
	Config.Transcoder.FFMPEG = f.FFMPEG
	Config.Transcoder.FFPROBE = f.FFPROBE
	Config.Transcoder.FFPROBE_TIMEOUT = (f.FFPROBE_TIMEOUT)
	Config.Transcoder.SEGMENT_TIME = f.SEGMENT_TIME
	Config.Transcoder.REQUEST_TIMEOUT = (f.REQUEST_TIMEOUT)
	NewConfig.Transcoder = Config.Transcoder
	ReWriteConfig()
	return nil
}
func DeleteStorage(name string) error {
	for i, storage := range NewConfig.Locations {
		if storage.Name == name {
			NewConfig.Locations = append(NewConfig.Locations[:i], NewConfig.Locations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("storage not found (may be already deleted)")
}
func CreateStorage(name string, storage_type string, options map[string]string) error {
	store, err := DispatchStorage(storage_type)
	if err != nil {
		return err
	}
	channel := make(chan error)
	go store.Init(name, channel, options, []kosmixutil.PathElement{})
	err = <-channel
	if err != nil {
		return err
	}
	store.Close()
	NewConfig.Locations = append(NewConfig.Locations, StorageElement{
		TYPE:    storage_type,
		Options: options,
		Paths:   []kosmixutil.PathElement{},
		Name:    name,
	})
	return nil
}
func AddPath(name string, AddPath kosmixutil.PathElement) error {
	for _, storage := range NewConfig.Locations {
		if storage.Name == name {
			for _, path := range storage.Paths {
				if AddPath.Path == path.Path {
					return fmt.Errorf("path already exists")
				}
			}
			storage.Paths = append(storage.Paths, AddPath)
			return nil
		}
	}
	return errors.New("storage not found")
}
func DeletePath(name string, DeletePath kosmixutil.PathElement) error {
	for _, storage := range NewConfig.Locations {
		if storage.Name == name {
			for i, path := range storage.Paths {
				if DeletePath.Path == path.Path {
					storage.Paths = append(storage.Paths[:i], storage.Paths[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("path not found")
		}
	}
	return fmt.Errorf("storage not found")
}

func ReWriteConfig() {
	f, err := os.OpenFile("config.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(NewConfig); err != nil {
		panic(err)
	}
}

func GetMaxSize(of string) int64 {
	if of == Movie {
		return Config.Limits.MovieSize
	}
	if of == Tv {
		return Config.Limits.SeasonSize
	}
	panic("Invalid type" + of)
}

func IsSsl() bool {
	return Config.Cert.Cert != "" && Config.Cert.Key != ""
}

func GetPrivateUrl() string {
	if IsSsl() {
		return "https://localhost:" + Config.Web.PublicPort
	}
	return "http://localhost:" + Config.Web.PublicPort
}

func ParseIdProvider(data string) (string, int, error) {
	var provider string
	var id int
	elements := strings.Split(data, "@")
	if len(elements) != 2 {
		return "", -1, fmt.Errorf("invalid id")
	}
	provider = strings.TrimSpace(elements[0])
	id, err := strconv.Atoi(elements[1])
	if err != nil {
		return "", -1, err
	}
	return provider, id, nil
}
func Get_movie_via_provider(provider_data string, create_if_not_exist bool, preload func() *gorm.DB) (*MOVIE, error) {
	var movie MOVIE
	provider, id, err := ParseIdProvider(provider_data)
	if err != nil {
		return nil, fmt.Errorf("invalid id")
	}
	switch provider {
	case "tmdb":
		tempMovie, err := InsertMovieInDb(db, id, -1, create_if_not_exist, preload)
		movie = *tempMovie
		if err != nil {
			return &movie, fmt.Errorf("error while getting movie")
		}
	case "db":
		if tx := preload().Where("id = ?", id).First(&movie); tx.Error != nil {
			return &movie, fmt.Errorf("movie not found")
		}
	default:
		return &movie, fmt.Errorf("invalid provider")
	}
	return &movie, nil
}
func Get_tv_via_provider(provider_data string, create_if_not_exist bool, preload func() *gorm.DB) (*TV, error) {
	var tv TV
	provider, id, err := ParseIdProvider(provider_data)
	if err != nil {
		return nil, fmt.Errorf("invalid id")
	}
	switch provider {
	case "tmdb":
		tempTv, err := GetSerieDb(db, id, "", create_if_not_exist, preload)
		tv = *tempTv
		if err != nil {
			return &tv, fmt.Errorf("error while getting tv show")
		}
	case "db":
		if tx := preload().Where("id = ?", id).First(&tv); tx.Error != nil {
			return &tv, fmt.Errorf("tv show not found \"" + strconv.Itoa(id) + "\"")
		}
	default:
		return &tv, fmt.Errorf("invalid provider" + provider)
	}
	return &tv, nil
}
func GetUser(db *gorm.DB, ctx *gin.Context, preload []string) (User, error) {
	var user User
	sess := sessions.Default(ctx)
	tx := db.Where("id = ?", sess.Get("user_id"))

	for _, p := range preload {
		tx = tx.Preload(p)
	}
	if tx.First(&user).Error != nil {
		return user, fmt.Errorf("user not found")
	}
	return user, nil
}
func GetUserWs(db *gorm.DB, user_id string, preload []string) (User, error) {
	var user User
	tx := db.Where("id = ?", user_id)
	for _, p := range preload {
		tx = tx.Preload(p)
	}
	if tx.First(&user).Error != nil {
		return user, fmt.Errorf("user not found")
	}
	return user, nil
}

// get quality by index
func GetQuality(i int) *QUALITY {
	for _, q := range Config.Transcoder.Qualitys {
		if q.Resolution == i {
			return &q
		}
	}
	return nil
}

func GetQualityByResolution(res int) *QUALITY {
	for _, q := range Config.Transcoder.Qualitys {
		if q.Resolution == res {
			return &q
		}
	}
	return nil
}
