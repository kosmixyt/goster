package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anacrolix/torrent"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

type FILE struct {
	gorm.Model
	ID         uint     `gorm:"unique;not null,primary_key"` // use
	TORRENT_ID uint     `gorm:"default:null"`
	TORRENT    *Torrent `gorm:"foreignKey:TORRENT_ID"`
	MOVIE_ID   uint     `gorm:"default:null"`
	MOVIE      *MOVIE   `gorm:"foreignKey:MOVIE_ID"`
	EPISODE_ID uint     `gorm:"default:null"`
	EPISODE    *EPISODE `gorm:"foreignKey:EPISODE_ID"`
	SEASON_ID  uint     `gorm:"default:null"`
	SEASON     *SEASON  `gorm:"foreignKey:SEASON_ID"`
	TV_ID      uint     `gorm:"default:null"`
	TV         *TV      `gorm:"foreignKey:TV_ID"`
	IS_MEDIA   bool     `gorm:"not null"`
	FILENAME   string   `gorm:"not null"`
	SUB_PATH   string   `gorm:"not null"`
	// if storage is null, then it is the torrent download path
	// STORAGE        *StorageDbElement
	StoragePathElement *StoragePathElement `gorm:"foreignKey:STORAGE_ID"`
	STORAGE_ID         *uint               `gorm:"default:null"`
	SIZE               int64               `gorm:"not null"`
	WATCHING           []WATCHING          `gorm:"foreignKey:FILE_ID;constraint:OnDelete:CASCADE"`
	SHARES             []Share             `gorm:"foreignKey:FILE_ID;constraint:OnDelete:CASCADE"`
	SourceRecord       *Record             `gorm:"foreignKey:SourceRecordID"`
	SourceRecordID     *uint               `gorm:"default:null"`
}

func (f *FILE) GetMediaType() string {
	if f.MOVIE_ID != 0 {
		return Movie
	}
	if f.TV_ID != 0 {
		return Tv
	}
	return ""
}
func (f *FILE) GetMediaId() int {
	if f.MOVIE_ID != 0 {
		return int(f.MOVIE_ID)
	}
	if f.TV_ID != 0 {
		return int(f.TV_ID)
	}
	return 0
}
func (f *FILE) LoadPath() *StoragePathElement {
	if f.StoragePathElement == nil {
		db.Model(f).Preload("StoragePathElement").First(f)
	}
	if f.StoragePathElement == nil {
		fmt.Println("StoragePathElement is still nil", f.FILENAME)
	}

	return f.StoragePathElement
}

func (f *FILE) GetPath(absolute bool) string {

	if absolute {
		offset := ""
		if f.LoadPath() == nil {
			offset = Config.Torrents.DownloadPath
		} else {
			offset = f.LoadPath().Path
		}
		return Joins(offset, f.SUB_PATH, f.FILENAME)
	}
	return Joins(f.SUB_PATH, f.FILENAME)
}

func (f *FILE) stats() (fs.FileInfo, error) {
	storer, err := f.LoadStorage()
	if err != nil {
		return os.Stat(f.GetPath(true))
	}
	return storer.toConn().Stats(f.GetPath(true))
}

func (f *FILE) LoadStorage() (*StorageDbElement, error) {
	f.LoadPath()
	if f.StoragePathElement == nil {
		return nil, errors.New("storage ID is nil")
	}
	return f.StoragePathElement.getStorage(), nil
}
func (f *FILE) MoveStorage(target_store *MemoryStorage) error {
	store, err := f.LoadStorage()
	if err != nil {
		if f.IsTorrentFile() {
			return errors.New("cannot move torrent file")
		}
		panic("if file is not torrent should load storage")
	}
	if store.ID == target_store.DbElement.ID {
		return errors.New("source and target storage are the same")
	}
	if f.isThisFileSourceOfCurrentConvert() {
		return errors.New("cannot move file while it is being converted")
	}
	if f.isThisFileSourceOfTranscoder() {
		return errors.New("cannot move file while it is being transcoded")
	}
	if f.IsTorrentFile() {
		return errors.New("cannot move torrent file")
	}
	return nil
}
func (f *FILE) GetReader() (io.ReadSeekCloser, error) {
	if f.TORRENT_ID != 0 {
		return f.GetFileInTorrent().NewReader(), nil
	}
	store, err := f.LoadStorage()
	if err != nil {
		panic("if file is torrent f.TORRENT_ID should not be 0")
	}
	file, err := store.toConn().GetReader(f.GetPath(true))
	if err != nil {
		return nil, err
	}
	return file, nil
}
func (f *FILE) Ffurl(app *gin.Engine) string {
	storer, err := f.LoadStorage()
	if err != nil {
		fmt.Println("streaming torrent file")
		if f.isCompleted() {
			return f.GetPath(true)
		}
		return GetFfUrl(app, f)
	}

	ffurl, needProxy := storer.toConn().GetFfmpegUrl(f.GetPath(true))
	if needProxy {
		return GetFfUrl(app, f)
	}
	return ffurl
}
func (f *FILE) GetDownloadUrl() string {
	base_url := Config.Web.PublicUrl + "/download?fileId=" + fmt.Sprint(f.ID)

	return base_url
}
func (f *FILE) GetTranscodeUrl() string {
	base_url := Config.Web.PublicUrl + "/transcode?fileId=" + fmt.Sprint(f.ID)
	return base_url
}
func (f *FILE) ClearFromTranscoder(operation string) {
	for _, transcoder := range Transcoders {
		if v, ok := transcoder.Source.(*FILE); ok {
			if v.ID == f.ID {
				transcoder.Destroy("File " + operation)
			}
		}
	}
}
func (f *FILE) EpisodeNumber() int {
	h, _ := kosmixutil.GetEpisode(f.FILENAME)
	return h
}
func (f *FILE) SeasonNumber() int {
	h, _ := kosmixutil.GetSeason(f.FILENAME, f.SUB_PATH)
	return h
}
func (f *FILE) Quality() string {
	return kosmixutil.GetQuality(f.FILENAME)
}
func (f *FILE) Codec() string {
	return kosmixutil.GetCodec(f.FILENAME)
}
func (f *FILE) Source() string {
	return kosmixutil.GetSource(f.FILENAME)
}
func (f *FILE) IsEpisode() bool {
	return kosmixutil.GetType(f.FILENAME, f.SUB_PATH) == "episode"
}

func (f *FILE) GetTitle() string {
	regexConvert := regexp.MustCompile(`convert-[0-9]{2,4}-`)
	b_ := regexConvert.ReplaceAll([]byte(f.FILENAME), []byte(""))
	return kosmixutil.GetTitle(string(b_))
}

func (f *FILE) IsTorrentFile() bool {
	return f.TORRENT_ID != 0
}
func (f *FILE) isCompleted() bool {
	// return f.TORRENT_ID == 0 || f.GetFileInTorrent().BytesCompleted() >= f.SIZE && isThisFileAnOutputOfConvert(f) == nil
	fmt.Println("f.TORRENT_ID", f == nil)
	completed := f.TORRENT_ID == 0
	if f.TORRENT_ID != 0 {
		completed = f.GetFileInTorrent().BytesCompleted() >= f.SIZE
	}
	return completed
}
func (f *FILE) GetFileInTorrent() *torrent.File {
	if !f.IsTorrentFile() {
		fmt.Println("[WARN] GetFileInTorrent called on non torrent file")
		return nil
	}
	torrent := GetTorrent(f.TORRENT_ID)
	if torrent == nil {
		panic("Torrent not found")
	}
	for i := 0; i < len(torrent.Torrent.Files()); i++ {
		fileinto := torrent.Torrent.Files()[i]
		if (f.SIZE) == fileinto.Length() && f.FILENAME == filepath.Base(fileinto.Path()) {
			return fileinto
		}
	}
	fmt.Println("f.FILENAME", f.FILENAME, torrent.Torrent.Name(), torrent.DB_ID)
	panic("File not found in torrent" + f.FILENAME)
}

func (f *FILE) FfprobeData(app *gin.Engine) (*FFPROBE_DATA, error) {
	args := []string{}
	url := f.Ffurl(app)
	if !f.isCompleted() {
		url := GetFfUrl(app, f)
		if strings.HasPrefix(url, "http") {
			args = []string{
				"-tls_verify", "0",
			}
		}
	}
	args = append(args, []string{
		"-hide_banner", "-loglevel", "fatal", "-show_error",
		"-show_format",
		"-show_streams",
		"-show_private_data",
		"-print_format", "json",
		url}...)
	stdout, err := exec.Command(Config.Transcoder.FFPROBE, args...).Output()
	if err != nil {
		return &FFPROBE_DATA{}, errors.New("Error running ffprobe : " + err.Error())
	}
	var ffprobeData FFPROBE_DATA
	if err = json.Unmarshal(stdout, &ffprobeData); err != nil {
		return &FFPROBE_DATA{}, err
	}
	return &ffprobeData, nil
}

func (f *FILE) IsBrowserPlayable() bool {
	fmt.Println("f.FILENAME", f.FILENAME)
	return strings.HasSuffix(f.FILENAME, "mp4") || strings.HasSuffix(f.FILENAME, "webm")
}

func (f *FILE) GetWatching(user *User, preload func() *gorm.DB) *WATCHING {
	var watch WATCHING
	if err := preload().Where("user_id = ? AND file_id = ?", user.ID, f.ID).First(&watch).Error; err != nil {
		watch = WATCHING{
			CURRENT:  0,
			TOTAL:    0,
			FILE:     *f,
			FILE_ID:  f.ID,
			USER:     user,
			USER_ID:  user.ID,
			MOVIE_ID: f.MOVIE_ID,
		}
		if f.EPISODE_ID != 0 {
			watch.EPISODE_ID = &f.EPISODE_ID
			watch.TV_ID = f.TV_ID
		}
		err := db.Create(&watch).Error
		if err != nil {
			panic(err)
		}
		if err := preload().Where("id = ?", watch.ID).First(&watch).Error; err != nil {
			panic(err)
		}
		return &watch
	}
	return &watch
}
func (f *FILE) isSameFile(file *FILE) bool {
	return f.ID == file.ID
}
func (f *FILE) isThisFileSourceOfCurrentConvert() bool {
	for _, convert := range Converts {
		if f.isSameFile(convert.SOURCE_FILE) {
			return true
		}
	}
	return false
}
func (f *FILE) isThisFileSourceOfTranscoder() bool {
	for _, transcoder := range Transcoders {
		file, isFileSource := transcoder.Source.(*FILE)
		if isFileSource {
			if f.isSameFile(file) {
				return true
			}
		}
	}
	return false
}

// Rename file !!only filename
func (f *FILE) Rename(filename string) error {
	if f.IsTorrentFile() {
		return errors.New("cannot rename torrent file")
	}
	if f.isThisFileSourceOfCurrentConvert() {
		return errors.New("cannot rename file while it is being converted")
	}
	if f.isThisFileSourceOfTranscoder() {
		return errors.New("cannot rename file while it is being transcoded")
	}
	storer, err := f.LoadStorage()
	if err != nil {
		panic("if file is not torrent should load storage")
	}
	storer.toConn().Rename(f.GetPath(true), Joins(f.LoadPath().Path, f.SUB_PATH, filename))
	db.Model(f).Update("filename", filename)
	return nil
}
func (f *FILE) LoadMedia(user *User) {
	if f.MOVIE_ID != 0 {
		user.SkinnyMoviePreloads().Preload("WATCHING", "USER_ID = ? ", user.ID).Where("id = ?", f.MOVIE_ID).Find(&f.MOVIE)
	} else if f.TV_ID != 0 {
		user.SkinnyTvPreloads().Preload("WATCHING", "USER_ID = ? ", user.ID).Where("id = ?", f.TV_ID).Find(&f.TV)
	}
}

func (f *FILE) SkinnyRender(user *User) SKINNY_RENDER {
	f.LoadMedia(user)
	if f.MOVIE_ID != 0 {
		return f.MOVIE.Skinny(f.MOVIE.GetWatching())
	}
	if f.TV_ID != 0 {
		return f.TV.Skinny(f.TV.GetWatching())
	}
	panic("file is not movie or tv")
}

func Joins(paths ...string) string {
	path := filepath.Join(paths...)
	return strings.ReplaceAll(path, "\\", "/")
}
