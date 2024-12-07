package engine

import (
	"fmt"
	"strconv"
	"sync"

	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

type TV struct {
	gorm.Model
	ID                uint `gorm:"unique;not null,primary_key"`
	TMDB_ID           int  `gorm:"not null"`
	ALTERNATIVE_NAMES string
	NAME              string     `gorm:"not null"`
	ORIGINAL_NAME     string     `gorm:"not null"`
	YEAR              int        `gorm:"not null"`
	SEASON            []*SEASON  `gorm:"foreignKey:TV_ID"`
	PROVIDERS         []PROVIDER `gorm:"many2many:tv_providers;"`
	FILES             []FILE     `gorm:"foreignKey:TV_ID"`
	TAGLINE           string     `gorm:"not null"`
	VIEW              int        `gorm:"not null,default:0"`
	// nombre de téléchargement d'épisode
	DOWNLOAD int `gorm:"not null,default:0"`
	// nombre de streaming d'épisode
	STREAMING                   int     `gorm:"not null,default:0"`
	DESCRIPTION                 string  `gorm:"not null"`
	Director                    string  `gorm:"not null"`
	Writer                      string  `gorm:"not null"`
	Awards                      string  `gorm:"not null"`
	Vote_average                float64 `gorm:"not null"`
	RATED                       string  `gorm:"not null"`
	RUNTIME                     int     `gorm:"not null"`
	VOTE                        float64 `gorm:"not null"`
	GENRE                       []GENRE `gorm:"many2many:tv_genres;"`
	BACKDROP_IMAGE_PATH         string  `gorm:"not null"`
	BACKDROP_IMAGE_STORAGE_TYPE int     `gorm:"not null"`
	POSTER_IMAGE_PATH           string  `gorm:"not null"`
	POSTER_IMAGE_STORAGE_TYPE   int     `gorm:"not null"`

	LOGO_IMAGE_PATH string `gorm:"not null"`
	// 1 = tmdb, 0 = local path, 2 = null
	LOGO_IMAGE_STORAGE_TYPE int `gorm:"not null"`
	// empty if no video trailer
	TRAILER_URL   string             `gorm:"not null"`
	WATCHLISTS    []User             `gorm:"many2many:watch_list_tvs;"`
	WATCHING      []WATCHING         `gorm:"foreignKey:TV_ID"`
	KEYWORDS      []KEYWORD          `gorm:"many2many:tv_keywords;"`
	REQUESTS      []*DownloadRequest `gorm:"foreignKey:TV_ID"`
	TORRENT_FILES []Torrent_File     `gorm:"foreignKey:TV_ID"`
}

func (t *TV) GetExistantSeasonById(id uint) *SEASON {
	for _, s := range t.SEASON {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (t *TV) GetNextEpisode(episode *EPISODE) *EPISODE {
	if episode.SEASON == nil {
		panic("season not preloaded")
	}
	// episode_number := episode.NUMBER
	// season_number := episode.SEASON.NUMBER
	// s := t.GetExistantSeasonById(episode.SEASON.ID)
	return nil
}

func (t *TV) GetSeason(season int, createIfNotExist bool) *SEASON {
	if len(t.SEASON) == 0 {
		fmt.Println("[WARN] No season found for TV", t.ID)
	}
	for _, s := range t.SEASON {
		if s.NUMBER == season {
			return s
		}
	}
	if !createIfNotExist {
		return nil
	}
	seasonElement := &SEASON{
		NUMBER:                      season,
		TV_ID:                       t.ID,
		DESCRIPTION:                 fmt.Sprintf("Description of season %d", season),
		NAME:                        fmt.Sprintf("Season %d", season),
		BACKDROP_IMAGE_PATH:         "",
		BACKDROP_IMAGE_STORAGE_TYPE: 2,
	}
	db.Save(seasonElement)
	t.SEASON = append(t.SEASON, seasonElement)
	return seasonElement
}
func (t *TV) GetFile(fileId uint) *FILE {
	for _, f := range t.FILES {
		if f.ID == fileId {
			return &f
		}
	}
	return nil
}
func (t *TV) IdString() string {
	return "db@" + strconv.Itoa(int(t.ID))
}
func (tv *TV) Render(user *User) TVItem {

	item := TVItem{
		ID:           "db@" + strconv.Itoa(int(tv.ID)),
		TMDB_ID:      int(tv.TMDB_ID),
		DISPLAY_NAME: tv.NAME,
		YEAR:         tv.YEAR,
		FILES:        []FileItem{},
		PROVIDERS:    ParseProviderItem(tv.PROVIDERS),
		AWARDS:       tv.Awards,
		DIRECTOR:     tv.Director,
		WRITER:       tv.Writer,
		Vote_average: tv.Vote_average,
		TAGLINE:      tv.TAGLINE,
		TYPE:         "tv",
		SIMILARS:     MapTvSkinny(tv.Similars(user.RenderMoviePreloads, 10)),
		DESCRIPTION:  tv.DESCRIPTION,
		RUNTIME:      tv.RUNTIME,
		TRAILER:      tv.TRAILER_URL,
		GENRE:        ParseGenreItem(tv.GENRE),
		SEASONS:      tv.ToSeason(),
	}
	switch tv.BACKDROP_IMAGE_STORAGE_TYPE {
	case 1:
		item.BACKDROP = TMDB_ORIGINAL + tv.BACKDROP_IMAGE_PATH
	case 0:
		item.BACKDROP = fmt.Sprintf(Config.Web.PublicUrl+"/image?type=tv&id=%d&image=backdrop", tv.TMDB_ID)
	}
	switch tv.POSTER_IMAGE_STORAGE_TYPE {
	case 1:
		item.POSTER = TMDB_ORIGINAL + tv.POSTER_IMAGE_PATH
	case 0:
		item.POSTER = fmt.Sprintf(Config.Web.PublicUrl+"/image?type=tv&id=%d&image=poster", tv.TMDB_ID)
	}
	switch tv.LOGO_IMAGE_STORAGE_TYPE {
	case 1:
		item.LOGO = TMDB_ORIGINAL + tv.LOGO_IMAGE_PATH
	case 0:
		item.LOGO = fmt.Sprintf(Config.Web.PublicUrl+"/image?type=tv&id=%d&image=logo", tv.TMDB_ID)
	}

	return item
}

func (tv *TV) Skinny(w *WATCHING) SKINNY_RENDER {
	URL := TMDB_LOW
	render := SKINNY_RENDER{

		ID:          "db@" + strconv.Itoa(int(tv.ID)),
		TYPE:        "tv",
		NAME:        tv.NAME,
		POSTER:      URL + tv.POSTER_IMAGE_PATH,
		BACKDROP:    URL + tv.BACKDROP_IMAGE_PATH,
		WATCH:       w.WatchData(),
		DESCRIPTION: tv.DESCRIPTION,
		TRAILER:     tv.TRAILER_URL,
		YEAR:        tv.YEAR,
		WATCHLISTED: len(tv.WATCHLISTS) > 0,
		RUNTIME:     strconv.Itoa(tv.RUNTIME),
		GENRE:       ParseGenreItem(tv.GENRE),
		LOGO:        URL + tv.LOGO_IMAGE_PATH,
		PROVIDERS:   ParseProviderItem(tv.PROVIDERS),
	}
	if w != nil {
		render.TRANSCODE_URL = Config.Web.PublicUrl + "/transcode?fileId=" + strconv.Itoa(int(w.FILE_ID))
	} else {
		render.TRANSCODE_URL = Config.Web.PublicUrl + "/transcode?type=tv&id=db@" + strconv.Itoa(int(tv.ID)) + "&season=1&episode=1"
	}
	return render
}
func (tv *TV) Similars(preload func() *gorm.DB, max int) []TV {
	var tvs []TV
	preload().
		Joins("JOIN tv_genres ON tvs.id = tv_genres.tv_id").
		Where("tv_genres.genre_id IN (?)", tv.GenreIds()).
		Order("RAND()").
		Limit(max).
		Find(&tvs)
	return tvs
}
func (tv *TV) GenreIds() []uint {
	genres := []uint{}
	for _, genre := range tv.GENRE {
		genres = append(genres, genre.ID)
	}
	return genres
}
func (tv *TV) DetermineEpisodeEnCoursDeLecture(watchings []WATCHING) *WATCHING {
	var good *WATCHING
	var user_id *uint = nil

	for _, watching := range watchings {
		if user_id == nil {
			user_id = &watching.USER_ID
		}
		if *user_id != watching.USER_ID || watching.TV_ID != tv.ID {
			panic("Invalid user")
		}
		if good == nil {
			good = &watching
			continue
		}
		if good.EPISODE.SEASON.ID == 0 {
			panic("season not preloaded")
		}
		if good.EPISODE.SEASON.NUMBER < watching.EPISODE.SEASON.NUMBER {
			good = &watching
		}
		if good.EPISODE.SEASON.NUMBER == watching.EPISODE.SEASON.NUMBER && good.EPISODE.NUMBER < watching.EPISODE.NUMBER {
			good = &watching
		}
	}

	return nil
}

func (tv *TV) GetWatchData() WatchData {
	return WatchData{}
}
func (tv *TV) GetWatching() *WATCHING {
	// get episode to render
	if len(tv.WATCHING) == 0 {
		return nil
	}
	return tv.DetermineEpisodeEnCoursDeLecture(tv.WATCHING)
}
func MapTvSkinny(tv []TV) []SKINNY_RENDER {
	render := make([]SKINNY_RENDER, 0)
	for _, t := range tv {
		render = append(render, t.Skinny(nil))
	}
	return render
}

func (tv *TV) ToSeason() []SeasonItem {
	if len(tv.SEASON) == 0 {
		fmt.Println("[WARN] No season found for TV", tv.ID)
		return []SeasonItem{}
	}
	seasons := []SeasonItem{}
	from := SortSeasonByNumber(tv.SEASON)
	for _, season := range from {
		newItemSeason := SeasonItem{
			ID:            season.ID,
			SEASON_NUMBER: season.NUMBER,
			NAME:          season.NAME,
			DESCRIPTION:   season.DESCRIPTION,
			EPISODES:      season.ToEpisode(),
		}
		switch season.BACKDROP_IMAGE_STORAGE_TYPE {
		case 1:
			newItemSeason.BACKDROP = TMDB_HIGH + season.BACKDROP_IMAGE_PATH
		case 0:
			newItemSeason.BACKDROP = fmt.Sprintf(Config.Web.PublicUrl+"/image?type=tv&id=%d&season=%d&image=backdrop", tv.TMDB_ID, season.NUMBER)
		}
		seasons = append(seasons, newItemSeason)
	}
	return seasons
}

func (tv *TV) Backdrop(base string) string {
	if tv.POSTER_IMAGE_STORAGE_TYPE != 2 {
		return base + tv.POSTER_IMAGE_PATH
	}
	return ""
}
func (tv *TV) Poster(base string) string {
	if tv.POSTER_IMAGE_STORAGE_TYPE != 2 {
		return base + tv.POSTER_IMAGE_PATH
	}
	return ""
}

func GetSerieDb(db *gorm.DB, serie int, year string, InitIfNotExist bool, preload func() *gorm.DB) (*TV, error) {
	insertItemMutex.Lock()
	defer insertItemMutex.Unlock()
	var serieInDb *TV
	if err := preload().Where("tmdb_id = ?", serie).First(&serieInDb).Error; err != nil {
		if !InitIfNotExist {
			return &TV{}, fmt.Errorf("serie not found in database")
		}
		serieData, err := kosmixutil.GetFullSerie(serie, year)
		if err != nil || serieData.ID == 0 || serieData.Name == "" {
			panic("Error while getting serie data")
		}
		runtime := 0
		year := serieData.First_air_date
		if len(year) > 4 {
			year = year[:4]
		}
		nyear, err := strconv.Atoi(year)
		if err != nil {
			fmt.Println("Cannot convert release date to int")
			panic(err)
		}
		serieInDb = &TV{
			TMDB_ID:                     serieData.ID,
			NAME:                        serieData.Name,
			ORIGINAL_NAME:               serieData.Original_name,
			YEAR:                        nyear,
			DESCRIPTION:                 serieData.Overview,
			TAGLINE:                     serieData.Tagline,
			RUNTIME:                     runtime,
			GENRE:                       ParseGenre(serieData.Genres, db),
			VOTE:                        serieData.Vote_average,
			PROVIDERS:                   ParseProvider(append(serieData.WatchProviders.Results.FR.Buy, serieData.WatchProviders.Results.FR.Rent...), db),
			POSTER_IMAGE_PATH:           "",
			POSTER_IMAGE_STORAGE_TYPE:   2,
			RATED:                       serieData.Rated,
			Director:                    serieData.Director,
			Writer:                      serieData.Writer,
			Awards:                      serieData.Awards,
			Vote_average:                serieData.Vote_average,
			BACKDROP_IMAGE_PATH:         "",
			BACKDROP_IMAGE_STORAGE_TYPE: 2,
			LOGO_IMAGE_PATH:             "",
			LOGO_IMAGE_STORAGE_TYPE:     2,
			TRAILER_URL:                 "",
		}
		if trailers, err := kosmixutil.GetVideo(serieData.Video.Results); err == nil {
			serieInDb.TRAILER_URL = "https://youtube.com/watch?v=" + trailers.Key
		}
		if logos, err := kosmixutil.GetImage(serieData.Images.Logo, kosmixutil.TMDB_IMAGE_LOGO_RATIO); err == nil {
			serieInDb.LOGO_IMAGE_PATH = logos.FilePath
			serieInDb.LOGO_IMAGE_STORAGE_TYPE = 1
		}
		if backdrops, err := kosmixutil.GetImage(serieData.Images.Backdrops, []float64{kosmixutil.TMDB_IMAGE_BACKDROP_RATIO}); err == nil {
			serieInDb.BACKDROP_IMAGE_PATH = backdrops.FilePath
			serieInDb.BACKDROP_IMAGE_STORAGE_TYPE = 1
		}
		if posters, err := kosmixutil.GetImage(serieData.Images.Posters, []float64{kosmixutil.TMDB_IMAGE_POSTER_RATIO}); err == nil {
			serieInDb.POSTER_IMAGE_PATH = posters.FilePath
			serieInDb.POSTER_IMAGE_STORAGE_TYPE = 1
		}
		db.Create(&serieInDb)
		seasonNumbers := []int{}
		for _, season := range serieData.Seasons {
			seasonNumbers = append(seasonNumbers, season.Season_number)
		}
		SeasonsData, err := kosmixutil.GetFullEpisodes(serieData.ID, seasonNumbers)
		if err != nil {
			panic("Error while getting serie data" + err.Error())
		}
		CreateSeason := func(db *gorm.DB, data kosmixutil.TMDB_FULL_SEASON, serieId uint, wg *sync.WaitGroup, channel chan *SEASON) {
			defer wg.Done()
			currentseason := &SEASON{
				NAME:                        data.Name,
				NUMBER:                      data.Season_number,
				DESCRIPTION:                 data.Overview,
				TV_ID:                       serieId,
				BACKDROP_IMAGE_PATH:         "",
				BACKDROP_IMAGE_STORAGE_TYPE: 2,
			}
			if data.Poster_path != "" {
				currentseason.BACKDROP_IMAGE_PATH = data.Poster_path
				currentseason.BACKDROP_IMAGE_STORAGE_TYPE = 1
			}
			CreateEpisode := func(db *gorm.DB, episode kosmixutil.TMDB_FULL_EPISODE, wg *sync.WaitGroup, channel chan *EPISODE) {
				defer wg.Done()
				currentepisode := &EPISODE{
					NAME:                     episode.Name,
					DESCRIPTION:              episode.Overview,
					SEASON_ID:                currentseason.ID,
					NUMBER:                   episode.Episode_number,
					STILL_IMAGE_PATH:         "",
					STILL_IMAGE_STORAGE_TYPE: 2,
				}
				if episode.Still_path != "" {
					currentepisode.STILL_IMAGE_PATH = episode.Still_path
					currentepisode.STILL_IMAGE_STORAGE_TYPE = 1
				}
				db.Create(&currentepisode)
				channel <- currentepisode
			}
			db.Create(&currentseason)
			var Waitgroup sync.WaitGroup
			channelsEpisodes := make(chan *EPISODE, len(data.Episodes))
			for _, episode := range data.Episodes {
				Waitgroup.Add(1)
				go CreateEpisode(db, episode, &Waitgroup, channelsEpisodes)
			}
			Waitgroup.Wait()
			for i := 0; i < len(data.Episodes); i++ {
				currentseason.EPISODES = append(currentseason.EPISODES, <-channelsEpisodes)
			}
			channel <- currentseason
		}
		var Waitgroup sync.WaitGroup
		seasonChannel := make(chan *SEASON, len(SeasonsData))
		for i := 0; i < len(SeasonsData); i++ {
			Waitgroup.Add(1)
			go CreateSeason(db, SeasonsData[i], serieInDb.ID, &Waitgroup, seasonChannel)
		}
		Waitgroup.Wait()
		for i := 0; i < len(SeasonsData); i++ {
			serieInDb.SEASON = append(serieInDb.SEASON, <-seasonChannel)
		}
		// db.Preload("SEASONS").Preload("SEASON.EPISODES")
	}
	return serieInDb, nil
}
