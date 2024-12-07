package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

var IPTV_IMG_PATH string
var IPTV_M3U8_PATH string
var FILES_TORRENT_PATH string
var HLS_OUTPUT_PATH string
var TRAILER_OUTPUT_PATH string
var FFMPEG_BIG_FILE_PATH string

var Config = AppConfig{}

func SetupCachePaths(cache_path string) {
	abs, err := filepath.Abs(cache_path)
	if err != nil {
		panic(err)
	}
	IPTV_IMG_PATH = Joins(abs, "iptv_imgs")
	IPTV_M3U8_PATH = Joins(abs, "iptv")
	FILES_TORRENT_PATH = Joins(abs, "torrents")
	HLS_OUTPUT_PATH = Joins(abs, "hls")
	TRAILER_OUTPUT_PATH = Joins(abs, "trailer")
	FFMPEG_BIG_FILE_PATH = Joins(abs, "ffmpeg")
	PanicOnError(os.MkdirAll(cache_path, os.ModePerm))
	PanicOnError(os.MkdirAll(IPTV_IMG_PATH, os.ModePerm))
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

var QUALITYS = []QUALITY{
	{
		Name:         "1080p-extra",
		Resolution:   1080,
		Width:        1920,
		VideoBitrate: 4000,
		AudioBitrate: 320,
	},
	{
		Name:         "1080p",
		Resolution:   1080,
		Width:        1920,
		VideoBitrate: 3500,
		AudioBitrate: 200,
	},
	{
		Name:         "720p",
		Resolution:   720,
		Width:        1280,
		VideoBitrate: 3000,
		AudioBitrate: 128,
	},
	{
		Name:         "480p",
		Resolution:   480,
		Width:        854,
		VideoBitrate: 2500,
		AudioBitrate: 128,
	},
	{
		Name:         "360p",
		Resolution:   360,
		Width:        640,
		VideoBitrate: 1000,
		AudioBitrate: 128,
	},
}

type StorageElement struct {
	TYPE    string      `json:"type"`
	Options interface{} `json:"options"`
	Name    string      `json:"name"`
}

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
		EnableForWebPlayableFiles bool   `json:"enable_for_web_playable_files"`
		MaxTranscoderThreads      int    `json:"max_transcoder_threads"`
		MaxConverterThreads       int    `json:"max_converter_threads"`
		FFMPEG                    string `json:"ffmpeg"`
		FFPROBE                   string `json:"ffprobe"`
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
	TorrentProviders struct {
		Sharewood struct {
			Key      string `json:"key"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"sharewood"`
		YGG struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"ygg"`
	} `json:"torrent_providers"`
}

func LoadConfig() {
	f, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&Config); err != nil {
		panic(err)
	}
	kosmixutil.InitKeys(Config.Metadata.Tmdb, Config.Metadata.Omdb, Config.Metadata.TmdbImgLang, Config.Metadata.TmdbLang)
	SetupCachePaths(Config.CachePath)
}

const FFPROBE_TIMEOUT = 10 * time.Second
const SEGMENT_TIME = 2.0
const AVANCE = 1
const THREAD_NUMBER = "4"

// les interval minimales entre les 2 requete
const LAST_REQUEST_TIMEOUT = 100
const REQUEST_TIMEOUT = SEGMENT_TIME * 8 * 1000 * time.Millisecond

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

// get quality by index
func GetQuality(i int) *QUALITY {
	for _, q := range QUALITYS {
		if q.Resolution == i {
			return &q
		}
	}
	return nil
}

func GetQualityByResolution(res int) *QUALITY {
	for _, q := range QUALITYS {
		if q.Resolution == res {
			return &q
		}
	}
	return nil
}
