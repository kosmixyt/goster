package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID                    uint                  `gorm:"unique;not null,primary_key"` // use
	NAME                  string                `gorm:"not null"`
	EMAIL                 string                `gorm:"unique"`
	TOKEN                 string                `gorm:"unique;not null"`
	ADMIN                 bool                  `gorm:"not null,default:false"`
	CAN_DOWNLOAD          bool                  `gorm:"not null"`
	CAN_CONVERT           bool                  `gorm:"not null"`
	CAN_ADD_FILES         bool                  `gorm:"not null"`
	SHARES                []Share               `gorm:"foreignKey:OWNER_ID"`
	CAN_UPLOAD            bool                  `gorm:"not null"`
	CAN_DELETE            bool                  `gorm:"not null"`
	CAN_EDIT              bool                  `gorm:"not null"`
	CAN_TRANSCODE         bool                  `gorm:"not null"`
	MAX_TRANSCODING       int                   `gorm:"not null"`
	TRANSCODING           int                   `gorm:"not null"`
	ALLOWED_UPLOAD_NUMBER int64                 `gorm:"not null"`
	CURRENT_UPLOAD_NUMBER int64                 `gorm:"not null"`
	CURRENT_UPLOAD_SIZE   int64                 `gorm:"not null"`
	ALLOWED_UPLOAD_SIZE   int64                 `gorm:"not null"`
	REAL_UPLOAD_SIZE      int64                 `gorm:"not null"`
	TORRENTS              []Torrent             `gorm:"foreignKey:USER_ID"`
	WATCH_LIST_MOVIES     []MOVIE               `gorm:"many2many:watch_list_movies;"`
	WATCH_LIST_TVS        []TV                  `gorm:"many2many:watch_list_tvs;"`
	WATCHING              []WATCHING            `gorm:"foreignKey:USER_ID"`
	IPTVS                 []IptvItem            `gorm:"foreignKey:USER_ID"`
	Tasks                 []*Task               `gorm:"foreignKey:USER_ID"`
	Requests              []DownloadRequest     `gorm:"foreignKey:OWNER_ID"`
	MediaQualityProfiles  []MediaQualityProfile `gorm:"foreignKey:UserID"`
	MediaQualitys         []MediaQuality        `gorm:"foreignKey:UserID"`
}

func (user *User) GetConverts() []*Convert {
	var converts []*Convert
	for _, convert := range Converts {
		if convert.User.ID == user.ID {
			converts = append(converts, convert)
		}
	}
	return converts
}

func (user *User) GetTask(id int) *Task {
	for _, task := range user.Tasks {
		if int(task.ID) == id && user.ID == task.USER_ID {
			// upgrade to local task
			if GetRuntimeTask(task.ID) != nil {
				return GetRuntimeTask(task.ID)
			}
			return task
		}
	}
	return nil
}
func (user *User) RemoveDeleteCredit(size int64) {
	db.Model(user).Updates(User{
		CURRENT_UPLOAD_NUMBER: user.CURRENT_UPLOAD_NUMBER - 1,
		CURRENT_UPLOAD_SIZE:   user.CURRENT_UPLOAD_SIZE - size,
	})
}

func (user *User) CreateTask(name string, cancel_func func() error) *Task {
	ts := &Task{
		Name:      name,
		on_Cancel: cancel_func,
		Logs:      "--- Start Logs ---\n",
		Status:    "PENDING",
		Started:   nil,
		Finished:  nil,
		User:      user,
		USER_ID:   user.ID,
	}
	Tasks = append(Tasks, ts)
	user.Tasks = append(user.Tasks, ts)
	db.Save(user)
	return ts
}

func (user *User) GetUserChannel(channel_id int) *IptvChannel {
	for _, iptv := range Iptvs {
		if iptv.USER_ID == user.ID || iptv.public() {
			channel := iptv.GetChannel(channel_id)
			if channel != nil {
				return channel
			}
		}
	}
	return nil
}
func (user *User) HaveUploadRight() bool {
	return user.CAN_UPLOAD
}
func (user *User) GetUserIptv() []*IptvItem {
	items := make([]*IptvItem, 0)
	for _, iptv := range Iptvs {
		if iptv.USER_ID == user.ID || iptv.public() {
			items = append(items, iptv)
		}
	}
	return items
}
func (user *User) GetIptvById(id int) *IptvItem {
	fmt.Println("len", len(Iptvs))
	for _, iptv := range Iptvs {
		fmt.Println("iptv", iptv.ID, iptv.USER_ID, user.TOKEN, id)
		if (iptv.ID == uint(id) && iptv.USER_ID == user.ID) || (iptv.public() && iptv.ID == uint(id)) {
			return iptv
		}
	}
	return nil
}
func (user *User) GetTorrents() []*GlTorrentItem {
	var torrents []*GlTorrentItem
	for _, item := range TORRENT_ITEMS {
		if item.OWNER_ID == user.ID {
			torrents = append(torrents, item)
		}
	}
	return torrents
}

func (user *User) GetUserTorrent(uuid uint) *GlTorrentItem {
	for _, item := range TORRENT_ITEMS {
		if item.OWNER_ID == 0 {
			panic("Owner id is 0")
		}
		if (item.DB_ID) == uuid && item.OWNER_ID == user.ID {
			return item
		}
	}
	return nil
}
func (user *User) IptvOrderedList() []*OrderedIptv {
	items := make([]*OrderedIptv, 0)
	for _, iptv := range user.GetUserIptv() {
		gr := make([]string, 0)
		for _, gp := range iptv.Groups {
			gr = append(gr, gp.Name)
		}
		items = append(items, &OrderedIptv{
			ID:                 iptv.ID,
			FileName:           iptv.FileName,
			MaxStreamCount:     iptv.MaxStreamCount,
			Groups:             gr,
			CurrentStreamCount: iptv.CurrentStreamCount,
			ChannelCount:       int64(len(iptv.Channels)),
		})

	}
	return items
}
func (user *User) GetShareId(id int) *Share {
	for _, share := range user.SHARES {
		if share.ID == uint(id) && share.OWNER_ID == user.ID {
			return &share
		}
	}
	return nil
}

func (user *User) GetUserTranscoders() []*Transcoder {
	transcoders := make([]*Transcoder, 0)
	for _, transcode := range Transcoders {
		if transcode.OWNER_ID == user.ID {
			transcoders = append(transcoders, transcode)
		}
	}
	return transcoders
}
func (user *User) GetTranscode(uuid string) *Transcoder {
	for _, transcode := range Transcoders {
		if transcode.OWNER_ID == user.ID && uuid == transcode.UUID {
			return transcode

		}
	}
	return nil
}
func (user *User) NewShare(expire *time.Duration, file FILE) *Share {
	share := &Share{
		OWNER_ID: user.ID,
		FILE_ID:  file.ID,
		EXPIRE:   time.Now().Add(*expire),
	}
	db.Create(share)
	return share
}
func (user *User) RenderMoviePreloads() *gorm.DB {
	return db.Preload("FILES").
		Preload("GENRE").
		Preload("PROVIDERS").
		Preload("FILES.WATCHING", "USER_ID = ?", user.ID).
		Preload("WATCHLISTS", "USER_ID = ?", user.ID)

}
func (user *User) SkinnyMoviePreloads() *gorm.DB {
	return db.Preload("PROVIDERS").
		Preload("GENRE").
		Preload("WATCHING", "user_id= ? ", user.ID).
		Preload("WATCHLISTS", "id= ? ", user.ID)
}
func (user *User) SkinnyTvPreloads() *gorm.DB {
	return db.Preload("PROVIDERS").
		Preload("GENRE").
		Preload("WATCHING", "user_id = ? ", user.ID).
		Preload("WATCHLISTS", "id= ? ", user.ID).
		Preload("WATCHING.TV").
		Preload("WATCHING.EPISODE").
		Preload("WATCHING.EPISODE.SEASON")

}
func (user *User) RenderTvPreloads() *gorm.DB {
	return db.Preload("GENRE").
		Preload("PROVIDERS").
		Preload("FILES.WATCHING", "USER_ID = ?", user.ID).
		Preload("SEASON").
		Preload("SEASON.EPISODES").
		Preload("SEASON.EPISODES.FILES").
		Preload("SEASON.EPISODES.FILES.WATCHING", "USER_ID = ?", user.ID).
		Preload("SEASON.EPISODES.WATCHING", "USER_ID = ?", user.ID)
}
func (user *User) Add_upload(size int64) {
	db.Model(user).Updates(map[string]interface{}{
		"CURRENT_UPLOAD_NUMBER": gorm.Expr("CURRENT_UPLOAD_NUMBER + 1"),
		"CURRENT_UPLOAD_SIZE":   gorm.Expr("CURRENT_UPLOAD_SIZE + ?", size),
	})
}
func (user *User) CanUpload(size int64) bool {
	return user.CURRENT_UPLOAD_SIZE+size <= user.ALLOWED_UPLOAD_SIZE && user.CURRENT_UPLOAD_NUMBER+1 <= user.ALLOWED_UPLOAD_NUMBER
}
func (user *User) HaveOneUploadCredit() bool {
	return user.CURRENT_UPLOAD_NUMBER+1 <= user.ALLOWED_UPLOAD_NUMBER
}
func (user *User) CAN_TRANSCODE_FILE() bool {
	return user.CAN_TRANSCODE && user.TRANSCODING < user.MAX_TRANSCODING
}
func (user *User) AddWatchListMovie(movie MOVIE) {
	for _, element := range user.WATCH_LIST_MOVIES {
		if element.ID == movie.ID {
			return
		}
	}
	user.WATCH_LIST_MOVIES = append(user.WATCH_LIST_MOVIES, movie)
	db.Save(user)
}
func (user *User) AddWatchListTv(tv TV) {
	for _, element := range user.WATCH_LIST_TVS {
		if element.ID == tv.ID {
			return
		}
	}
	user.WATCH_LIST_TVS = append(user.WATCH_LIST_TVS, tv)
	db.Save(user)
}
func (user *User) RemoveWatchListMovie(movie MOVIE) {
	indexOf := slices.IndexFunc(user.WATCH_LIST_MOVIES, func(m MOVIE) bool {
		return m.ID == movie.ID
	})
	if indexOf == -1 {
		return
	}
	db.Model(user).Association("WATCH_LIST_MOVIES").Delete(&movie)
}
func (user *User) RemoveWatchListTv(tv TV) {
	indexOf := slices.IndexFunc(user.WATCH_LIST_TVS, func(t TV) bool {
		return t.ID == tv.ID
	})
	if indexOf == -1 {
		return
	}
	db.Model(user).Association("WATCH_LIST_TVS").Delete(&tv)
}

// func (user *User) HttpDownload(url string, progress func(int64, int64), output_filename string, output_path string, o_mid *uint, o_tvid *uint, o_eid *uint, o_sid *uint) (*FILE, error) {
// 	if !user.CAN_UPLOAD {
// 		return nil, fmt.Errorf("User cannot upload")
// 	}
// 	req, err := http.NewRequest("GET", url, nil)

// 	if err != nil {
// 		return nil, err
// 	}
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
// 	http.DefaultClient.Timeout = time.Minute * 60
// 	resp, err := http.DefaultClient.Do(req)
// 	if output_filename == "" {
// 		output_filename = resp.Header.Get("Content-Disposition")
// 		if output_filename == "" {
// 			return nil, fmt.Errorf("no filename")
// 		}
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("error downloading file")
// 	}
// 	file, err := os.Create(filepath.Join(output_path, output_filename))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	var written int64
// 	var total int64
// 	total, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
// 	if err != nil {
// 		total = 0
// 	}
// 	buf := make([]byte, 1024)
// 	last_percent := 0
// 	for {
// 		n, err := resp.Body.Read(buf)
// 		if err != nil {
// 			if err.Error() == "EOF" {
// 				break
// 			}
// 		}
// 		written += int64(n)
// 		file.Write(buf[:n])
// 		percent := int(float64(written) / float64(total) * 100)
// 		if percent != last_percent {
// 			last_percent = percent
// 			progress(written, total)
// 		}
// 	}
// 	fileInDb := &FILE{FILENAME: output_path, PATH: output_path, SIZE: written, IS_MEDIA: true}
// 	if o_mid != nil {
// 		fileInDb.MOVIE_ID = *o_mid
// 	}
// 	if o_tvid != nil {
// 		fileInDb.TV_ID = *o_tvid
// 		fileInDb.SEASON_ID = *o_sid
// 		fileInDb.EPISODE_ID = *o_eid
// 	}
// 	db.Save(fileInDb)
// 	return fileInDb, nil
// }

func (user *User) GetReworkedWatching() []WATCHING {
	var WATCHINGS []WATCHING = make([]WATCHING, 0)
	if tx := db.
		Order("updated_at desc").
		Preload("MOVIE").
		Preload("MOVIE.PROVIDERS").
		Preload("MOVIE.GENRE").
		Preload("MOVIE.WATCHLISTS", "id = ? ", user.ID).
		Preload("TV").
		Preload("TV.GENRE").
		Preload("TV.PROVIDERS").
		Preload("TV.WATCHLISTS", "id = ? ", user.ID).
		Preload("EPISODE").
		Preload("EPISODE.SEASON").
		Where("user_id = ?", user.ID).Find(&WATCHINGS).Error; tx != nil {
		panic("Error while getting watchings" + tx.Error())
	}
	var episodesItems map[uint]*WATCHING = make(map[uint]*WATCHING)
	fmt.Println("watchings", len(WATCHINGS))
	finalWatchings := make([]WATCHING, 0)
wloop:
	for _, w := range WATCHINGS {
		if w.MOVIE != nil {
			if w.CURRENT < 60*5 || w.CURRENT > w.TOTAL-(60*5) {
				continue wloop
			}
			finalWatchings = append(finalWatchings, w)
		}
		if w.TV != nil {
			if episodesItems[w.TV_ID] == nil {
				episodesItems[w.TV_ID] = &w
				continue
			}
			if episodesItems[w.TV.ID].EPISODE.SEASON.NUMBER < w.EPISODE.SEASON.NUMBER {
				episodesItems[w.TV_ID] = &w
				continue
			}
			if episodesItems[w.TV.ID].EPISODE.SEASON.NUMBER == w.EPISODE.SEASON.NUMBER && episodesItems[w.TV.ID].EPISODE.NUMBER < w.EPISODE.NUMBER {
				episodesItems[w.TV_ID] = &w
				continue
			}
		}
	}
	for _, w := range episodesItems {
		finalWatchings = append(finalWatchings, *w)
	}
	sort.SliceStable(finalWatchings, func(i, j int) bool { return finalWatchings[i].UpdatedAt.After(finalWatchings[j].UpdatedAt) })
	return finalWatchings
}
func GetRecent(db *gorm.DB, user User) *Line_Render {
	RecentMovies, RecentTVs := make([]MOVIE, 1), make([]TV, 1)
	reqMovie := db.
		Order("created_at desc").
		Preload("PROVIDERS").
		Preload("GENRE").
		Where("BACKDROP_IMAGE_STORAGE_TYPE != 2").
		Preload("WATCHING", "user_id = ? ", user.ID).
		Preload("WATCHLISTS", "id = ? ", user.ID).
		Limit(15)
	reqSerie := db.
		Order("created_at desc").
		Preload("PROVIDERS").
		Preload("GENRE").
		Where("BACKDROP_IMAGE_STORAGE_TYPE != 2").
		Preload("WATCHING", "user_id = ? ", user.ID).
		Preload("WATCHLISTS", "id = ? ", user.ID).
		Limit(15)
	if reqSerie.Find(&RecentTVs).Error != nil || reqMovie.Find(&RecentMovies).Error != nil {
		panic("error while getting recent movies or series")
	}
	renderers, _ := make([]SKINNY_RENDER, 0), TMDB_ORIGINAL
	for _, m := range RecentMovies {
		renderers = append(renderers, m.Skinny(m.GetWatching(), &TMDB_ORIGINAL))
	}
	for _, t := range RecentTVs {
		renderers = append(renderers, t.Skinny(nil))
	}
	return &Line_Render{
		Data:  renderers,
		Title: "Recent TV Shows",
		Type:  "items",
	}
}

func (user *User) Most_Viewed(channel chan []Line_Render, wg *sync.WaitGroup) {
	defer wg.Done()
	movies, tvs, renderers := make([]MOVIE, 0), make([]TV, 0), make([]SKINNY_RENDER, 0)
	reqMovie := user.SkinnyMoviePreloads().
		Order("view desc").
		Limit(20)
	reqTv := user.SkinnyTvPreloads().
		Order("view desc").
		Limit(20)
	reqMovie.Find(&movies)
	reqTv.Find(&tvs)
	renderers = append(renderers, MapMovieSkinny(movies)...)
	renderers = append(renderers, MapTvSkinny(tvs)...)
	channel <- []Line_Render{
		Line_Render{
			Data:  renderers,
			Title: "Most Viewed",
			Type:  "items",
		},
	}
}
func (user *User) GetMovieRequest(id uint) *DownloadRequest {
	for _, request := range user.Requests {
		if request.MOVIE_ID == id {
			return &request
		}
	}
	return nil
}
func (user *User) GetTvRequest(id uint, SEASON_ID uint) *DownloadRequest {
	for _, request := range user.Requests {
		if request.TV_ID == id && request.TV_SEASON_ID == SEASON_ID {
			return &request
		}
	}
	return nil
}

func (user *User) GetWatchList() ([]MOVIE, []TV) {
	tvs, movies := make([]TV, 0), make([]MOVIE, 0)
	if user.SkinnyMoviePreloads().
		Where("id IN (SELECT movie_id FROM watch_list_movies WHERE user_id = ?)", user.ID).
		Find(&movies).Error != nil ||
		user.SkinnyTvPreloads().
			Where("id IN (SELECT tv_id FROM watch_list_tvs WHERE user_id = ?)", user.ID).
			Find(&tvs).Error != nil {
		panic("error while getting watchlist items")
	}
	return movies, tvs
}

func (user *User) Get_Liked_Genres() []uint {
	return []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
}

func (user *User) GetBestRated() ([]MOVIE, []TV) {
	movies, tvs := make([]MOVIE, 0), make([]TV, 0)
	if user.SkinnyMoviePreloads().Limit(15).Order("vote_average desc").Find(&movies).Error != nil || user.SkinnyTvPreloads().Order("vote_average desc").Limit(15).Find(&tvs).Error != nil {
		panic("error while getting best rated movies")
	}
	return movies, tvs
	// renderers = append(renderers, MapMovieSkinny(movies)...)
	// renderers = append(renderers, MapTvSkinny(tvs)...)
	// channel <- []Line_Render{
	// 	Line_Render{
	// 		Data:  renderers,
	// 		Title: "Best Rated",
	// 		Type:  "items",
	// 	},
	// }

}
func (user *User) AddIptv(file io.Reader, maxStreamCount int64) (*IptvItem, error) {
	output_name := fmt.Sprintf("%s.m3u8", time.Now().Format("2006-01-02-15-04-05"))
	f, err := os.Create(filepath.Join(IPTV_M3U8_PATH, output_name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	io.Copy(f, file)
	iptv := &IptvItem{
		USER_ID:            user.ID,
		MaxStreamCount:     maxStreamCount,
		CurrentStreamCount: 0,
		Channels:           make([]*IptvChannel, 0),
		RECORDS:            make([]*Record, 0),
		TranscodeIds:       make([]string, 0),
		FileName:           output_name,
	}
	db.Create(iptv)
	return iptv, nil
}
func (user *User) Upload(Storer *MemoryStorage, outPath string, name string, total int64, MOVIE *MOVIE, EPISODE *EPISODE) (*Upload, error) {
	if !user.CanUpload(total) {
		return nil, fmt.Errorf("User cannot upload")
	}
	file, err := Storer.Conn.GetWriter(outPath)
	if err != nil {
		return nil, err
	}
	if !Storer.DbElement.HasRootPath(outPath) {
		return nil, fmt.Errorf("outpath is not a root path of Storer")
	}
	upl := &Upload{
		USER_ID:     user.ID,
		Name:        name,
		EPISODE:     EPISODE,
		MOVIE:       MOVIE,
		Storer_path: outPath,
		Storer:      Storer,
		Writer:      file,
		CURRENT:     0,
		TOTAL:       total,
	}
	db.Create(&upl)
	Uploads = append(Uploads, upl)
	return upl, nil
}

func (user *User) GetUpload(id uint) *Upload {
	for _, upload := range Uploads {
		if upload.ID == id && upload.USER_ID == user.ID {
			return upload
		}
	}
	return nil
}
