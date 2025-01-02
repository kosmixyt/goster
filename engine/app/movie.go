package engine

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/dlclark/regexp2"
	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

var insertItemMutex sync.Mutex

type MOVIE struct {
	gorm.Model
	ID            uint   `gorm:"unique;not null,primary_key"` // use
	TMDB_ID       int    `gorm:"not null"`
	NAME          string `gorm:"not null"`
	ORIGINAL_NAME string `gorm:"not null"`
	YEAR          int    `gorm:"not null"`
	VIEW          int    `gorm:"not null,default:0"`
	// nombre de téléchargement de film
	DOWNLOAD int `gorm:"not null,default:0"`
	// nombre de streaming de film
	STREAMING           int        `gorm:"not null,default:0"`
	AGE_LIMIT           string     `gorm:"not null,default:No age limit"`
	FILES               []FILE     `gorm:"foreignKey:MOVIE_ID"`
	BUDGET              string     `gorm:"not null,default:0"`
	AWARDS              string     `gorm:"not null"`
	Director            string     `gorm:"not null"`
	Writer              string     `gorm:"not null"`
	Vote_average        float64    `gorm:"not null"`
	Origin_country      string     `gorm:"not null"`
	DIRECTOR            string     `gorm:"not null"`
	PROVIDERS           []PROVIDER `gorm:"many2many:movie_providers;"`
	DESCRIPTION         string     `gorm:"not null"`
	RUNTIME             string     `gorm:"not null"`
	GENRE               []GENRE    `gorm:"many2many:movie_genres;"`
	TAGLINE             string     `gorm:"not null"`
	BACKDROP_IMAGE_PATH string     `gorm:"not null"`
	// 1 = tmdb, 0 = local path, 2 = null
	POSTER_IMAGE_PATH string `gorm:"not null"`
	// 1 = tmdb, 0 = local path, 2 = null

	LOGO_IMAGE_PATH string `gorm:"not null"`
	// 1 = tmdb, 0 = local path, 2 = null
	// empty if no video trailer
	TRAILER_URL string             `gorm:"not null"`
	WATCHLISTS  []User             `gorm:"many2many:watch_list_movies;"`
	WATCHING    []WATCHING         `gorm:"foreignKey:MOVIE_ID"`
	KEYWORDS    []KEYWORD          `gorm:"many2many:movie_keywords;"`
	REQUESTS    []*DownloadRequest `gorm:"foreignKey:MOVIE_ID"`
	Torrents    []*Torrent_File    `gorm:"foreignKey:MOVIE_ID"`
}

func (m *MOVIE) GetMediaType() string {
	return Movie
}
func (m *MOVIE) GetMediaId() int {
	return int(m.ID)
}
func (m *MOVIE) GetFiles() []FILE {
	return m.FILES
}
func (m *MOVIE) LoadFiles(db *gorm.DB) []FILE {
	db.Model(m).Association("FILES").Find(&m.FILES)
	return m.FILES
}

func (m *MOVIE) Refresh(preload func() *gorm.DB) {
	if err := preload().Where("id = ?", m.ID).First(&m).Error; err != nil {
		panic(err)
	}
}

func (m *MOVIE) HasFile(file *FILE) bool {
	if len(m.FILES) == 0 {
		fmt.Println("[WARN] No file found for movie", m.ID)
	}
	if file == nil {
		return len(m.FILES) > 0
	}
	for _, f := range m.FILES {
		if f.ID == file.ID {
			return true
		}
	}
	return false
}
func (m *MOVIE) GetFileId(id int) *FILE {
	for _, file := range m.FILES {
		if file.ID == uint(id) {
			return &file
		}
	}
	return nil
}
func (m *MOVIE) GetFile() *FILE {
	if len(m.FILES) == 0 {
		fmt.Println("[WARN] No file found for movie", m.ID)
	}
	return &m.FILES[0]
}
func (m *MOVIE) IdString() string {
	return "db@" + strconv.Itoa(int(m.ID))
}
func (movie *MOVIE) GetSearchName() []string {
	names := []string{}
	regexJapaneese, err := regexp2.Compile(`/[\u3000-\u303F]|[\u3040-\u309F]|[\u30A0-\u30FF]|[\uFF00-\uFFEF]|[\u4E00-\u9FAF]|[\u2605-\u2606]|[\u2190-\u2195]|\u203B/g`, 0)
	if err != nil {
		panic(err)
	}
	if have, _ := regexJapaneese.MatchString(movie.NAME); !have {
		names = append(names,
			movie.NAME+" "+strconv.Itoa(movie.YEAR),
			movie.NAME+" "+strconv.Itoa(movie.YEAR+1),
			movie.NAME+" "+strconv.Itoa(movie.YEAR-1),
		)
	}
	if have, _ := regexJapaneese.MatchString(movie.ORIGINAL_NAME); !have {
		names = append(names,
			movie.ORIGINAL_NAME+" "+strconv.Itoa(movie.YEAR),
			movie.ORIGINAL_NAME+" "+strconv.Itoa(movie.YEAR+1),
			movie.ORIGINAL_NAME+" "+strconv.Itoa(movie.YEAR-1),
		)
	}
	return names
}

func (MOVIE) GetMaxSize() int64 {
	return GetMaxSize(Movie)
}

// UPDATE movies SET `backdrop_image_storage_type` = 1 WHERE `backdrop_image_storage_type` = 0
// UPDATE movies SET `poster_image_storage_type` = 1 WHERE `poster_image_storage_type` = 0
func (m *MOVIE) GetPoster(quality string) ([]byte, error) {
	return GetMinia(
		m.POSTER_IMAGE_PATH,
		int(m.ID),
		Movie,
		"poster",
		quality,
	)
}
func (m *MOVIE) GetBackdrop(quality string) ([]byte, error) {
	return GetMinia(
		m.BACKDROP_IMAGE_PATH,
		int(m.ID),
		Movie,
		"backdrop",
		quality,
	)
}
func (m *MOVIE) GetLogo(quality string) ([]byte, error) {
	return GetMinia(
		m.LOGO_IMAGE_PATH,
		int(m.ID),
		Movie,
		"logo",
		quality,
	)
}

func emptyMovie(name string, year int) *MOVIE {
	return &MOVIE{
		NAME:                name,
		YEAR:                year,
		TMDB_ID:             -1,
		FILES:               []FILE{},
		DESCRIPTION:         "",
		RUNTIME:             "0",
		ORIGINAL_NAME:       "",
		DOWNLOAD:            0,
		STREAMING:           0,
		VIEW:                0,
		PROVIDERS:           []PROVIDER{},
		TRAILER_URL:         "",
		WATCHLISTS:          []User{},
		WATCHING:            []WATCHING{},
		KEYWORDS:            []KEYWORD{},
		GENRE:               []GENRE{},
		BACKDROP_IMAGE_PATH: "",
		LOGO_IMAGE_PATH:     "",
		POSTER_IMAGE_PATH:   "",
	}

}

func (movie *MOVIE) Render(user *User) MovieItem {
	if !movie.HasFile(nil) {
		fmt.Println("[WARN] No file found for movie (preload)", movie.ID)
	}
	if len(movie.GENRE) == 0 {
		fmt.Println("[WARN] No genre found for movie (preload)", movie.ID)
	}
	if len(movie.PROVIDERS) == 0 {
		fmt.Println("[WARN] No provider found for movie (preload)", movie.ID)
	}
	item := MovieItem{
		ID:            "db@" + strconv.Itoa(int(movie.ID)),
		DISPLAY_NAME:  movie.NAME,
		YEAR:          movie.YEAR,
		FILES:         movie.ToFile(),
		PROVIDERS:     ParseProviderItem(movie.PROVIDERS),
		BUDGET:        movie.BUDGET,
		AWARDS:        movie.AWARDS,
		DIRECTOR:      movie.DIRECTOR,
		WRITER:        movie.Writer,
		Vote_average:  movie.Vote_average,
		TAGLINE:       movie.TAGLINE,
		TYPE:          "movie",
		WATCHLISTED:   len(movie.WATCHLISTS) > 0,
		SIMILARS:      MapMovieSkinny(movie.SimilarMovies(user.SkinnyMoviePreloads, 10)),
		DESCRIPTION:   movie.DESCRIPTION,
		TRAILER:       movie.TRAILER_URL,
		RUNTIME:       movie.RUNTIME,
		GENRE:         ParseGenreItem(movie.GENRE),
		DOWNLOAD_URL:  fmt.Sprintf(Config.Web.PublicUrl+"/download?type=movie&id=db@%d", movie.ID),
		TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=movie&id=db@%d", movie.ID),
	}
	if len(movie.FILES) > 0 {
		if len(movie.FILES[0].WATCHING) > 1 {
			panic("More than one watching")
		}
		if len(movie.FILES[0].WATCHING) > 0 {
			item.WATCH = WatchData{TOTAL: movie.FILES[0].WATCHING[0].CURRENT, CURRENT: movie.FILES[0].WATCHING[0].TOTAL}
		}
	}
	item.LOGO = movie.Logo("high")
	item.BACKDROP = movie.Backdrop("high")
	item.POSTER = movie.Poster("high")
	return item
}

func (movie *MOVIE) Skinny(w *WATCHING) SKINNY_RENDER {
	render := SKINNY_RENDER{
		ID:          "db@" + strconv.Itoa(int(movie.ID)),
		TYPE:        "movie",
		NAME:        movie.NAME,
		POSTER:      movie.Poster("low"),
		BACKDROP:    movie.Backdrop("low"),
		DESCRIPTION: movie.DESCRIPTION,
		YEAR:        movie.YEAR,
		RUNTIME:     movie.RUNTIME,
		WATCHLISTED: len(movie.WATCHLISTS) > 0,
		TRAILER:     movie.TRAILER_URL,
		WATCH:       w.WatchData(),
		GENRE:       ParseGenreItem(movie.GENRE),
		PROVIDERS:   ParseProviderItem(movie.PROVIDERS),
		LOGO:        TMDB_LOW + movie.LOGO_IMAGE_PATH,
	}
	if w != nil {
		render.TRANSCODE_URL = fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=movie&id=db@%d", movie.ID)
	} else {
		render.TRANSCODE_URL = fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=movie&id=db@%d", movie.ID)
	}
	return render
}
func (movie *MOVIE) GetWatching() *WATCHING {
	if len(movie.WATCHING) == 0 {
		return nil
	}
	return &movie.WATCHING[0]
}
func MapMovieSkinny(movies []MOVIE) []SKINNY_RENDER {
	render := make([]SKINNY_RENDER, 0)
	for _, m := range movies {
		render = append(render, m.Skinny(m.GetWatching()))
	}
	return render
}

func (m *MOVIE) SimilarMovies(preloads func() *gorm.DB, max int) []MOVIE {
	if m == nil {
		panic("movie is nil")
	}
	if len(m.GenreIds()) == 0 {
		return make([]MOVIE, 0)
	}

	var movies []MOVIE
	preloads().
		Select("DISTINCT movies.*").
		Joins("JOIN movie_genres ON movies.id = movie_genres.movie_id").
		Where("movie_genres.genre_id IN (?)", m.GenreIds()).
		Where("movies.id != ?", m.ID).
		Limit(40).
		Find(&movies)
	return movies
}
func (m *MOVIE) GenreIds() []int {
	ids := []int{}
	for _, g := range m.GENRE {
		ids = append(ids, int(g.ID))
	}
	return ids
}

func (movie *MOVIE) ToFile() []FileItem {
	files := []FileItem{}
	for _, file := range movie.FILES {
		appnt := FileItem{
			ID:            file.ID,
			FILENAME:      file.FILENAME,
			SIZE:          file.SIZE,
			DOWNLOAD_URL:  file.GetDownloadUrl(),
			TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=movie&id=db@%d&fileId=%d", movie.ID, file.ID),
		}
		if file.WATCHING != nil {
			if len(file.WATCHING) > 0 {
				appnt.CURRENT = file.WATCHING[0].CURRENT
			}
		}
		files = append(files, appnt)
	}
	return files
}

func (movie *MOVIE) Backdrop(quality string) string {
	return fmt.Sprintf(Config.Web.PublicUrl+"/image?type=movie&id=db@%d&image=backdrop&quality=%s", movie.ID, quality)
}
func (movie *MOVIE) Poster(quality string) string {
	return fmt.Sprintf(Config.Web.PublicUrl+"/image?type=movie&id=db@%d&image=poster&quality=%s", movie.ID, quality)
}
func (movie *MOVIE) Logo(quality string) string {
	return fmt.Sprintf(Config.Web.PublicUrl+"/image?type=movie&id=db@%d&image=logo&quality=%s", movie.ID, quality)
}

func InsertMovieInDb(db *gorm.DB, movie int, year int64, InsertIfNotExist bool, preload func() *gorm.DB) (*MOVIE, error) {
	insertItemMutex.Lock()
	defer insertItemMutex.Unlock()
	var movieInDb MOVIE
	if tx := preload().Where("tmdb_id = ?", movie).First(&movieInDb); tx.Error != nil {
		movieData, err := kosmixutil.GetFullMovie(movie, year)
		if err != nil {
			panic(err)
		}
		if movieData.ID == 0 || movieData.Title == "" {
			return &MOVIE{}, fmt.Errorf("movie not found in tmdb")
		}
		year := movieData.Release_date
		if len(year) > 4 {
			year = year[:4]
		}
		nyear, err := strconv.Atoi(year)
		if err != nil {
			fmt.Println("Cannot convert release date to int")
			nyear = 0
		}
		movieInDb = MOVIE{
			DOWNLOAD:            0,
			STREAMING:           0,
			Origin_country:      movieData.Original_language,
			BUDGET:              strconv.Itoa(movieData.Budget),
			TAGLINE:             movieData.Tagline,
			AGE_LIMIT:           movieData.Rated,
			AWARDS:              movieData.Awards,
			DIRECTOR:            movieData.Director,
			Director:            movieData.Director,
			Writer:              movieData.Writer,
			Vote_average:        movieData.Vote_average,
			TMDB_ID:             movie,
			ORIGINAL_NAME:       movieData.Original_title,
			DESCRIPTION:         movieData.Overview,
			RUNTIME:             movieData.Runtime,
			VIEW:                0,
			NAME:                movieData.Title,
			YEAR:                nyear,
			PROVIDERS:           ParseProvider(append(movieData.WatchProviders.Results.FR.Buy, movieData.WatchProviders.Results.FR.Rent...), db),
			FILES:               []FILE{},
			GENRE:               ParseGenre(movieData.Genres, db),
			BACKDROP_IMAGE_PATH: "",

			POSTER_IMAGE_PATH: "",
			LOGO_IMAGE_PATH:   "",
			TRAILER_URL:       "",
		}
		if logos, err := kosmixutil.GetImage(movieData.Images.Logo, kosmixutil.TMDB_IMAGE_LOGO_RATIO); err == nil {
			movieInDb.LOGO_IMAGE_PATH = logos.FilePath
		}
		if trailers, err := kosmixutil.GetVideo(movieData.Videos.Results); err == nil {
			movieInDb.TRAILER_URL = "https://youtube.com/watch?v=" + trailers.Key
		}
		if posters, err := kosmixutil.GetImage(movieData.Images.Posters, []float64{kosmixutil.TMDB_IMAGE_POSTER_RATIO}); err == nil {
			movieInDb.POSTER_IMAGE_PATH = posters.FilePath
		}
		if backdrops, err := kosmixutil.GetImage(movieData.Images.Backdrops, []float64{kosmixutil.TMDB_IMAGE_BACKDROP_RATIO}); err == nil {
			movieInDb.BACKDROP_IMAGE_PATH = backdrops.FilePath
		}
		preload().Create(&movieInDb)
	}
	return &movieInDb, nil
}

func DeferedInsert(db *gorm.DB, movie int, year int64, InsertIfNotExist bool, preload func() *gorm.DB, channel chan *MOVIE, wg *sync.WaitGroup) {
	defer wg.Done()
	movieInDb, err := InsertMovieInDb(db, movie, year, InsertIfNotExist, preload)
	if err != nil {
		fmt.Println(err)
	}
	channel <- movieInDb
}

func ParseGenre(from []kosmixutil.GENRE, db *gorm.DB) []GENRE {
	genres := []GENRE{}
	for _, genre := range from {
		var genreInDb GENRE
		if tx := db.Where("id = ?", genre.ID).First(&genreInDb); tx.Error != nil {
			genreInDb = GENRE{
				NAME: genre.Name,
				ID:   genre.ID,
			}
			db.Create(&genreInDb)
		}
		genres = append(genres, genreInDb)
	}
	return genres
}
func ParseProvider(from []kosmixutil.TMDB_WATCH_PROVIDER, db *gorm.DB) []PROVIDER {
	providers := []PROVIDER{}
	for _, provider := range from {
		var providerInDb PROVIDER
		if tx := db.Where("prov_id_er_id = ?", provider.Provider_id).First(&providerInDb); tx.Error != nil {
			providerInDb = PROVIDER{
				PROVIDER_ID:      uint(provider.Provider_id),
				PROVIDER_NAME:    provider.Provider_name,
				LOGO_PATH:        provider.Logo_path,
				DISPLAY_PRIORITY: provider.Display_priority,
			}
			db.Create(&providerInDb)
		}
		providers = append(providers, providerInDb)
	}
	return providers
}
