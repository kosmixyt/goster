package engine

import (
	"time"

	"github.com/anacrolix/torrent"
	"gorm.io/gorm"
)

type GlTorrentItem struct {
	OWNER_ID       uint
	Torrent        *torrent.Torrent
	DB_ITEM        *Torrent
	START_DOWNLOAD int64
	MEDIA_UUID     uint
	MEDIA_TYPE     string
	START_UPLOAD   int64
	START          time.Time
}

type Record struct {
	gorm.Model
	ID                   uint                `gorm:"unique;not null,primary_key"`
	IPTV                 IptvItem            `gorm:"foreignKey:IPTV_ID"`
	IPTV_ID              uint                `gorm:"not null"`
	START                time.Time           `gorm:"not null"`
	DURATION             int64               `gorm:"not null"`
	OWNER                *User               `gorm:"foreignKey:OWNER_ID"`
	Force                bool                `gorm:"not null"`
	OWNER_ID             uint                `gorm:"not null"`
	OUTPUT_EPISODE       *EPISODE            `gorm:"foreignKey:OUTPUT_EPISODE_ID"`
	OUTPUT_MOVIE         *MOVIE              `gorm:"foreignKey:OUTPUT_MOVIE_ID"`
	OUTPUT_EPISODE_ID    *uint               `gorm:"default:null"`
	OUTPUT_MOVIE_ID      *uint               `gorm:"default:null"`
	CHANNEL_ID           int64               `gorm:"not null"`
	ERROR                string              `gorm:"default:null"`
	Task                 *Task               `gorm:"foreignKey:TASK_ID"`
	TASK_ID              uint                `gorm:"not null"`
	ENDED                bool                `gorm:"not null"`
	OutputStorerMem      *MemoryStorage      `gorm:"-"`
	OutputPathStorer     *StoragePathElement `gorm:"foreignKey:OutputPathStorerId"`
	OutputPathStorerId   uint                `gorm:"default:null"`
	OutputStorerFileName string              `gorm:"not null"`
	OutputFile           *FILE               `gorm:"foreignKey:OUTPUT_FILE_ID"`
	OUTPUT_FILE_ID       *uint               `gorm:"default:null"`
}
type QUALITY struct {
	Name              string  `json:"Name"`
	Resolution        int     `json:"Resolution"`
	Width             int     `json:"Width"`
	BitrateMultiplier float32 `json:"BitrateMultiplier"`
	VideoBitrate      int     `json:"VideoBitrate"`
	AudioBitrate      int     `json:"AudioBitrate"`
}

type GENRE struct {
	gorm.Model
	ID     uint    `gorm:"unique;not null,primary_key"`
	NAME   string  `gorm:"not null"`
	TVS    []TV    `gorm:"many2many:tv_genres;"`
	MOVIES []MOVIE `gorm:"many2many:movie_genres;"`
}

type KEYWORD struct {
	gorm.Model
	ID     uint    `gorm:"unique;not null,primary_key"` // use
	NAME   string  `gorm:"not null"`
	Movies []MOVIE `gorm:"many2many:movie_keywords;"`
	TVS    []TV    `gorm:"many2many:tv_keywords;"`
}
type KEYWORDitem struct {
	ID   uint
	NAME string
}

type Line_Render struct {
	Data  []SKINNY_RENDER
	Title string
	Type  string
}
type Provider_Line struct {
	Data  []PROVIDERItem
	Title string
	Type  string
}
type Api_Home struct {
	Top       Line_Render
	Lines     []Line_Render
	Providers []Line_Render
}

type IptvItem struct {
	gorm.Model
	ID                 uint `gorm:"unique;not null,primary_key"` // use
	USER               User `gorm:"foreignKey:USER_ID"`
	USER_ID            uint
	MaxStreamCount     int64
	CurrentStreamCount int64 `gorm:"-"`
	FileName           string
	Channels           []*IptvChannel `gorm:"-"`
	Groups             []*IptvGroup   `gorm:"-"`
	TranscodeIds       []string       `gorm:"-"`
	RECORDS            []*Record      `gorm:"foreignKey:IPTV_ID"`
}
type IptvGroup struct {
	Name     string
	Channels []*IptvChannel
}
type IptvChannel struct {
	Id           int64
	Name         string
	Logo_url     string
	Group        *IptvGroup
	Url          string
	Iptv         *IptvItem
	TranscodeIds []string `gorm:"-"`
}
type IptvChannelRender struct {
	Name          string
	Logo_url      string
	GroupName     string
	Id            int64
	TRANSCODE_URL string
}
type OrderedIptv struct {
	ID                 uint
	MaxStreamCount     int64
	CurrentStreamCount int64
	FileName           string
	Groups             []string
	ChannelCount       int64
}

type PROVIDER struct {
	gorm.Model
	PROVIDER_ID      uint    `gorm:"unique;not null,primary_key"`
	LOGO_PATH        string  `gorm:"not null"`
	PROVIDER_NAME    string  `gorm:"not null"`
	DISPLAY_PRIORITY int     `gorm:"not null"`
	MOVIES           []MOVIE `gorm:"many2many:movie_providers;"`
	TVS              []TV    `gorm:"many2many:tv_providers;"`
}

type GENERATED_TOKEN struct {
	gorm.Model
	ID      uint   `gorm:"unique;not null,primary_key"` // use
	USER_ID uint   `gorm:"not null"`
	IP      string `gorm:"not null"`
	TOKEN   string
}

type ALTERNATIVE_NAME struct {
	gorm.Model
	ID    uint `gorm:"unique;not null,primary_key"` // use
	TV_ID uint `gorm:"not null"`
	TV    TV   `gorm:"foreignKey:TV_ID"`
	NAME  string
}

type Torrent struct {
	gorm.Model
	ID                uint             `gorm:"unique;not null,primary_key"` // use
	USER_ID           uint             `gorm:"not null"`
	USER              User             `gorm:"foreignKey:USER_ID"` // use UUID
	PATH              string           `gorm:"not null"`
	FINISHED          bool             `gorm:"not null"`
	Progress          float64          `gorm:"not null"`
	Name              string           `gorm:"not null"`
	InfoHash          string           `gorm:"not null"`
	Size              int64            `gorm:"not null"`
	Paused            bool             `gorm:"not null"`
	DOWNLOAD          int64            `gorm:"not null"`
	UPLOAD            int64            `gorm:"not null"`
	TIME_TO_1_PERCENT float64          `gorm:"default:null"`
	FILES             []*FILE          `gorm:"foreignKey:TORRENT_ID"`
	DL_PATH           string           `gorm:"not null;default:wesh alors"`
	PROVIER_NAME      string           `gorm:"not null"`
	REQUESTS          *DownloadRequest `gorm:"foreignKey:TORRENT_ID"`
}

type AUDIO_TRACK struct {
	Index int    `json:"Index"`
	Name  string `json:"Name"`
}
type SUBTITLE struct {
	Index int    `json:"Index"`
	Name  string `json:"Name"`
}

type MovieItem struct {
	ID            string
	DISPLAY_NAME  string
	YEAR          int
	FILES         []FileItem
	BUDGET        string
	AWARDS        string
	DIRECTOR      string
	WRITER        string
	TAGLINE       string
	PROVIDERS     []PROVIDERItem
	WATCH         WatchData
	LOGO          string
	TYPE          string
	SIMILARS      []SKINNY_RENDER
	DESCRIPTION   string
	RUNTIME       string
	Vote_average  float64
	GENRE         []GenreItem
	BACKDROP      string
	POSTER        string
	TRAILER       string
	DOWNLOAD_URL  string
	TRANSCODE_URL string
	WATCHLISTED   bool
}
type WatchData struct {
	TOTAL      int64
	CURRENT    int64
	UPDATED_AT time.Time
}
type TVItem struct {
	ID           string
	TMDB_ID      int
	DISPLAY_NAME string
	LOGO         string
	YEAR         int
	TYPE         string
	AWARDS       string
	DIRECTOR     string
	WRITER       string
	Vote_average float64
	TAGLINE      string
	FILES        []FileItem
	PROVIDERS    []PROVIDERItem
	SIMILARS     []SKINNY_RENDER
	DESCRIPTION  string
	WATCHLISTED  bool
	RUNTIME      int
	GENRE        []GenreItem
	BACKDROP     string
	POSTER       string
	TRAILER      string
	SEASONS      []SeasonItem
}
type SeasonItem struct {
	ID            uint
	SEASON_NUMBER int
	NAME          string
	DESCRIPTION   string
	BACKDROP      string
	EPISODES      []EpisodeItem
}
type EpisodeItem struct {
	ID             uint
	EPISODE_NUMBER int
	FILES          []FileItem
	WATCH          WatchData
	NAME           string
	DESCRIPTION    string
	STILL          string
	DOWNLOAD_URL   string
	TRANSCODE_URL  string
}
type GenreItem struct {
	ID   uint
	NAME string
}
type FileItem struct {
	ID            uint
	FILENAME      string
	SIZE          int64
	DOWNLOAD_URL  string
	TRANSCODE_URL string
	CURRENT       int64
}
type PROVIDERItem struct {
	PROVIDER_ID      int
	URL              string
	PROVIDER_NAME    string
	DISPLAY_PRIORITY int
}

type SKINNY_RENDER struct {
	ID string
	// PROVIDER    string
	WATCH         WatchData
	TYPE          string
	NAME          string
	POSTER        string
	BACKDROP      string
	TRAILER       string
	DESCRIPTION   string
	YEAR          int
	RUNTIME       string
	GENRE         []GenreItem
	WATCHLISTED   bool
	TRANSCODE_URL string
	LOGO          string
	PROVIDERS     []PROVIDERItem
	DisplayData   string
}
type GenreClassement struct {
	Genre    GENRE
	Count    int
	ItemType string
	Items    []*ITEM_INF
}
type ITEM_INF struct {
	TMDB_ID int
	TYPE    string
	NAME    string
}

type TranscoderRes struct {
	Manifest          string        `json:"manifest"`
	Download_url      string        `json:"download_url"`
	Uuid              string        `json:"uuid"`
	Qualitys          []QUALITY     `json:"qualitys"`
	Tracks            []AUDIO_TRACK `json:"tracks"`
	Subtitles         []SUBTITLE    `json:"subtitles"`
	Current           int64         `json:"current"`
	Total             int64         `json:"total"`
	Seasons           []SeasonItem  `json:"seasons"`
	Name              string        `json:"name"`
	Poster            string        `json:"poster"`
	Backdrop          string        `json:"backdrop"`
	IsLive            bool          `json:"isLive"`
	Next              NextFile      `json:"next"`
	Task_id           uint          `json:"task_id"`
	IsBrowserPlayable bool          `json:"isBrowserPlayable"`
}
type NextFile struct {
	TYPE          string
	TRANSCODE_URL string
	DOWNLOAD_URL  string
	BACKDROP      string
	NAME          string
	FILENAME      string
}
