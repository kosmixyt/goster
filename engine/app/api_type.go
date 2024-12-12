package engine

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"time"

	"kosmix.fr/streaming/kosmixutil"
)

func TmdbSkinnyRender(movie *kosmixutil.TMDB_SEARCH_RESULT_MOVIE, tv *kosmixutil.TMDB_SEARCH_RESULT_SERIE, multi *kosmixutil.TMDB_MULTI_SEARCH_RESULT) SKINNY_RENDER {
	if movie != nil {
		intYear, err := strconv.Atoi(movie.Release_date)
		if err != nil {
			intYear = -1
		}
		return SKINNY_RENDER{
			ID:            "tmdb@" + strconv.Itoa(int(movie.ID)),
			TYPE:          "movie",
			NAME:          movie.Original_title,
			POSTER:        TMDB_LOW + movie.Poster_path,
			BACKDROP:      TMDB_LOW + movie.Backdrop_path,
			DESCRIPTION:   movie.Overview,
			TRAILER:       "",
			YEAR:          intYear,
			PROVIDERS:     make([]PROVIDERItem, 0),
			WATCH:         WatchData{TOTAL: 0, CURRENT: 0, UPDATED_AT: time.Now()},
			GENRE:         make([]GenreItem, 0),
			RUNTIME:       "0",
			WATCHLISTED:   false,
			LOGO:          "",
			TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=movie&id=tmdb@%d", movie.ID),
		}

	}
	if tv != nil {
		yearInt, err := strconv.Atoi(tv.First_air_date)
		if err != nil {
			fmt.Println("Error while parsing year: ", err)
			yearInt = -1
		}
		return SKINNY_RENDER{
			ID:            "tmdb@" + strconv.Itoa(tv.ID),
			TYPE:          "tv",
			NAME:          tv.Name,
			TRAILER:       "",
			POSTER:        TMDB_LOW + tv.Poster_path,
			BACKDROP:      TMDB_LOW + tv.Backdrop_path,
			DESCRIPTION:   tv.Overview,
			YEAR:          yearInt,
			RUNTIME:       "0",
			GENRE:         make([]GenreItem, 0),
			PROVIDERS:     make([]PROVIDERItem, 0),
			WATCH:         WatchData{TOTAL: 0, CURRENT: 0, UPDATED_AT: time.Now()},
			WATCHLISTED:   false,
			LOGO:          "",
			TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=tv&id=tmdb@%d", tv.ID),
		}
	}
	if multi != nil {
		yearInt, err := strconv.Atoi(multi.Release_date)
		if err != nil {
			fmt.Println("Error while parsing year: ", err)
			yearInt = -1
		}
		item := SKINNY_RENDER{
			ID:            "tmdb@" + strconv.Itoa(multi.ID),
			TYPE:          multi.Media_type,
			NAME:          multi.Original_title,
			TRAILER:       "",
			POSTER:        TMDB_LOW + multi.Poster_path,
			BACKDROP:      TMDB_LOW + multi.Backdrop_path,
			DESCRIPTION:   multi.Overview,
			GENRE:         make([]GenreItem, 0),
			PROVIDERS:     make([]PROVIDERItem, 0),
			YEAR:          yearInt,
			RUNTIME:       "0",
			WATCH:         WatchData{TOTAL: 0, CURRENT: 0, UPDATED_AT: time.Now()},
			WATCHLISTED:   false,
			LOGO:          "",
			TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=%s&id=tmdb@%d", multi.Media_type, multi.ID),
		}
		if item.NAME == "" {
			item.NAME = multi.Title
		}
		if item.NAME == "" {
			item.NAME = multi.Name
		}
		return item

	}
	panic("tmdbSkinnyRender: all nil")
}

func ParseProviderItem(from []PROVIDER) []PROVIDERItem {
	providers := []PROVIDERItem{}
	for _, provider := range from {
		providers = append(providers, PROVIDERItem{
			PROVIDER_ID:      int(provider.ID),
			URL:              TMDB_LOW + provider.LOGO_PATH,
			PROVIDER_NAME:    provider.PROVIDER_NAME,
			DISPLAY_PRIORITY: provider.DISPLAY_PRIORITY,
		})
	}
	return providers
}
func ParseKeywordItem(from []KEYWORD) []KEYWORDitem {
	keywords := []KEYWORDitem{}
	for _, keyword := range from {
		keywords = append(keywords, KEYWORDitem{
			ID:   keyword.ID,
			NAME: keyword.NAME,
		})
	}
	return keywords
}
func ParseGenreItem(from []GENRE) []GenreItem {
	genres := []GenreItem{}
	for _, genre := range from {
		genres = append(genres, GenreItem{
			ID:   (genre.ID),
			NAME: genre.NAME,
		})
	}
	return genres
}
func SortSeasonByNumber(from []*SEASON) []*SEASON {
	for i := 0; i < len(from); i++ {
		for j := 0; j < len(from)-1; j++ {
			if from[j].NUMBER > from[j+1].NUMBER {
				from[j], from[j+1] = from[j+1], from[j]
			}
		}
	}
	return from
}
func SortEpisodeByNumber(from []*EPISODE) []*EPISODE {
	for i := 0; i < len(from); i++ {
		for j := 0; j < len(from)-1; j++ {
			if from[j].NUMBER > from[j+1].NUMBER {
				from[j], from[j+1] = from[j+1], from[j]
			}
		}
	}
	return from
}

func GetMinia(poster_image_path string, source_id int, source_type string, wantedImageType string, quality string) ([]byte, error) {
	if !slices.Contains([]string{"poster", "backdrop", "logo"}, wantedImageType) {
		return nil, fmt.Errorf("invalid image type")
	}
	if !slices.Contains([]string{"low", "high"}, quality) {
		return nil, fmt.Errorf("invalid quality")
	}
	var base_url string
	switch quality {
	case "low":
		base_url = TMDB_LOW
	case "high":
		base_url = TMDB_HIGH
	}
	path := Joins(IMG_PATH, source_type+"_"+strconv.Itoa(int(source_id))+"_"+quality+"_"+wantedImageType+".png")
	if _, err := os.Stat(path); os.IsNotExist(err) {

		read, err := kosmixutil.DownloadImage(base_url, poster_image_path)
		if err != nil {
			return nil, err
		}
		fullPoster, err := io.ReadAll(read)
		if err != nil {
			return nil, err
		}
		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		// _, err = io.Copy(file, read)
		file.Write(fullPoster)
		if err != nil {
			return nil, err
		}
		return fullPoster, nil
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		fullpst, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		return fullpst, nil
	}

}

var ErrorIsNotAdmin = fmt.Errorf("user is not admin")
var ErrorInvalidAction = fmt.Errorf("invalid action")
var ErrorRescanFailed = fmt.Errorf("rescan failed")
var ErrorInvalidQuality = fmt.Errorf("invalid quality")
var ErrorInvalidMediaType = fmt.Errorf("invalid media type")
var ErrorInvalidImage = fmt.Errorf("invalid image")
var ErrorCannotRecord = fmt.Errorf("cannot record")
var ErrorChannelNotFound = fmt.Errorf("channel not found")
var ErrorEpisodeNotFound = fmt.Errorf("episode not found")
var ErrorMovieNotFound = fmt.Errorf("movie not found")
var ErrorInvalidOutputType = fmt.Errorf("invalid output_type")
var ErrorRecordNotFound = fmt.Errorf("record not found")
