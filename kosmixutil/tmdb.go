package kosmixutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var API_KEY string
var OMDB_API_KEY string

const API_URL = "https://api.themoviedb.org/3"
const TMDB_IMAGE_URL = "https://image.tmdb.org/t/p/original"

const TMDB_IMAGE_BACKDROP_RATIO = 1.778
const TMDB_IMAGE_POSTER_RATIO = 0.667

var TMDB_IMAGE_LOGO_RATIO = []float64{0}

var SHORT_LANGUAGE string = "fr"
var TMDB_IMAGE_PREFERED_LANGUAGE = []string{}

func InitKeys(tmdb_api_key string, omdb_api_key string, languages []string, lang string) {
	API_KEY = tmdb_api_key
	OMDB_API_KEY = omdb_api_key
	TMDB_IMAGE_PREFERED_LANGUAGE = languages
	SHORT_LANGUAGE = lang
}
func DownloadImage(base_path string, path string) (io.Reader, error) {
	url := base_path + path
	resp, err := http.Get(url)
	fmt.Println(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Error getting poster")
	}

	return resp.Body, nil
}

func SearchForSerie(name string, year string) (TMDB_SEARCH_SERIE, error) {
	var resp *http.Response = nil
	var err error = nil
	if year != "" {
		resp, err = http.Get(API_URL + "/search/tv?api_key=" + API_KEY + "&query=" + url.QueryEscape(name) + "&year=" + year + "&language=" + SHORT_LANGUAGE)
	} else {
		resp, err = http.Get(API_URL + "/search/tv?api_key=" + API_KEY + "&query=" + url.QueryEscape(name) + "&language=" + SHORT_LANGUAGE)
	}
	if err != nil {
		return TMDB_SEARCH_SERIE{}, err
	}
	var out TMDB_SEARCH_SERIE
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return TMDB_SEARCH_SERIE{}, err
	}
	return out, nil
}
func SearchForMovie(name string, year int) (TMDB_SEARCH_MOVIE, error) {
	var URL string
	if year != -1 {
		URL = API_URL + "/search/movie?api_key=" + API_KEY + "&query=" + url.QueryEscape(strings.TrimSpace(name)) + "&year=" + strconv.Itoa(year) + "&language=" + SHORT_LANGUAGE
	} else {
		URL = API_URL + "/search/movie?api_key=" + API_KEY + "&query=" + url.QueryEscape(strings.TrimSpace(name)) + "&language=" + SHORT_LANGUAGE
	}
	resp, err := http.Get(URL)
	if err != nil {
		return TMDB_SEARCH_MOVIE{}, err
	}
	var result TMDB_SEARCH_MOVIE
	json.NewDecoder(resp.Body).Decode(&result)
	defer resp.Body.Close()
	return result, nil
}
func MultiSearch(name string) (TMDB_MULTI_SEARCH, error) {
	url := API_URL + "/search/multi?api_key=" + API_KEY + "&query=" + url.QueryEscape(strings.TrimSpace(name)) + "&language=" + SHORT_LANGUAGE + "&include_adult=true"
	resp, err := http.Get(url)
	if err != nil {
		return TMDB_MULTI_SEARCH{}, err
	}
	defer resp.Body.Close()
	var result TMDB_MULTI_SEARCH
	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}
func GetFullMovie(id int, year int64) (METADATA_FULL_MOVIE, error) {
	url := API_URL + "/movie/" + fmt.Sprint(id) + "?api_key=" + API_KEY + "&language=" + SHORT_LANGUAGE + "&append_to_response=watch/providers,images,videos,external_ids&include_image_language=fr,en,null"
	if year != -1 {
		url += "&year=" + strconv.FormatInt(year, 10)
	}
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		return METADATA_FULL_MOVIE{}, err
	}
	defer resp.Body.Close()
	var result TMDB_FULL_MOVIE
	json.NewDecoder(resp.Body).Decode(&result)
	jsonData := METADATA_FULL_MOVIE{
		Adult:                 result.Adult,
		Backdrop_path:         result.Backdrop_path,
		Belongs_to_collection: result.Belongs_to_collection,
		Budget:                result.Budget,
		Genres:                result.Genres,
		Homepage:              result.Homepage,
		ID:                    result.ID,
		Imdb_id:               result.Imdb_id,
		Original_language:     result.Original_language,
		OriginCountry:         result.OriginCountry,
		Original_title:        result.Original_title,
		Overview:              result.Overview,
		Popularity:            result.Popularity,
		Poster_path:           result.Poster_path,
		Production_companies:  result.Production_companies,
		Production_countries:  result.Production_countries,
		Release_date:          result.Release_date,
		Revenue:               result.Revenue,
		Runtime:               strconv.Itoa(result.Runtime) + "$",
		Rated:                 "0",
		Director:              "",
		Awards:                "",
		BoxOffice:             "",
		Writer:                "",
		Spoken_languages:      result.Spoken_languages,
		Status:                result.Status,
		Tagline:               result.Tagline,
		Title:                 result.Title,
		Vote_average:          result.Vote_average,
		Vote_count:            result.Vote_count,
		WatchProviders:        result.WatchProviders,
		Images:                result.Images,
		Videos:                result.Videos,
		External_ids:          result.External_ids,
	}
	if result.External_ids.Imdb_id == "" || result.Imdb_id == "" {
		return jsonData, nil
	}
	var imdb_id string = result.External_ids.Imdb_id
	if imdb_id == "" {
		imdb_id = result.Imdb_id
	}
	if OMDB_API_KEY == "" {
		return jsonData, nil
	}
	url = "https://www.omdbapi.com/?apikey=" + OMDB_API_KEY + "&i=" + imdb_id
	resp, err = http.Get(url)
	if err != nil {
		fmt.Println("Error getting omdb", err)
		return jsonData, nil
	}
	defer resp.Body.Close()
	var omdbResult OMDB_FULL_ITEM
	json.NewDecoder(resp.Body).Decode(&omdbResult)
	jsonData.Awards = omdbResult.Awards
	jsonData.Rated = omdbResult.Rated
	jsonData.Director = omdbResult.Director
	jsonData.Writer = omdbResult.Writer
	jsonData.BoxOffice = omdbResult.BoxOffice
	floatImdbRating, err := strconv.ParseFloat(omdbResult.ImdbRating, 64)
	if err != nil {
		fmt.Println("Error parsing imdb rating", err)
	}
	jsonData.Vote_average = floatImdbRating
	return jsonData, nil
}

func GetFullSerie(id int, year string) (METADATA_FULL_TV, error) {
	// url := API_URL + "/search/tv?api_key=" + config.API_KEY + "&query=" + url.QueryEscape(name) + "&year=" + year + "&language=" + SHORT_LANGUAGE
	url := API_URL + "/tv/" + fmt.Sprint(id) + "?api_key=" + API_KEY + "&language=" + SHORT_LANGUAGE + "&append_to_response=,images,videos,external_ids,watch/providers,seasons&include_image_language=fr,en,null"
	if year != "" {
		url += "&year=" + year
	}
	resp, err := http.Get(url)
	if err != nil {
		return METADATA_FULL_TV{}, err
	}
	defer resp.Body.Close()
	var result TMDB_FULL_SERIE
	json.NewDecoder(resp.Body).Decode(&result)
	jsonData := METADATA_FULL_TV{
		Adult:                result.Adult,
		Backdrop_path:        result.Backdrop_path,
		Created_by:           result.Created_by,
		Runtime:              "0",
		First_air_date:       result.First_air_date,
		Genres:               result.Genres,
		Homepage:             result.Homepage,
		ID:                   result.ID,
		In_production:        result.In_production,
		Languages:            result.Languages,
		Last_air_date:        result.Last_air_date,
		Last_episode_to_air:  result.Last_episode_to_air,
		Director:             "",
		Writer:               "",
		Awards:               "",
		Name:                 result.Name,
		Next_episode_to_air:  result.Next_episode_to_air,
		Networks:             result.Networks,
		Rated:                "0",
		Number_of_episodes:   result.Number_of_episodes,
		Number_of_seasons:    result.Number_of_seasons,
		Origin_country:       result.Origin_country,
		Original_language:    result.Original_language,
		Original_name:        result.Original_name,
		Overview:             result.Overview,
		Popularity:           result.Popularity,
		Poster_path:          result.Poster_path,
		Production_companies: result.Production_companies,
		Production_countries: result.Production_countries,
		Seasons:              result.Seasons,
		Spoken_languages:     result.Spoken_languages,
		Status:               result.Status,
		Tagline:              result.Tagline,
		Type:                 result.Type,
		Vote_average:         result.Vote_average,
		Vote_count:           result.Vote_count,
		WatchProviders:       result.WatchProviders,
		Images:               result.Images,
		Video:                result.Video,
		External_ids:         result.External_ids,
	}
	if result.External_ids.Imdb_id == "" {
		return jsonData, nil
	}
	url = "https://www.omdbapi.com/?apikey=" + OMDB_API_KEY + "&i=" + result.External_ids.Imdb_id
	resp, err = http.Get(url)
	if err != nil {
		fmt.Println("Error getting omdb", err)
		return jsonData, nil
	}
	defer resp.Body.Close()
	var omdbResult OMDB_FULL_ITEM
	json.NewDecoder(resp.Body).Decode(&omdbResult)
	jsonData.Awards = omdbResult.Awards
	jsonData.Rated = omdbResult.Rated
	jsonData.Director = omdbResult.Director
	jsonData.Writer = omdbResult.Writer
	floatImdbRating, err := strconv.ParseFloat(omdbResult.ImdbRating, 64)
	if err != nil {
		fmt.Println("Error parsing imdb rating", err)
	}
	jsonData.Vote_average = floatImdbRating
	return jsonData, nil

}
func GetSimilarMovies(id int) ([]TMDB_SEARCH_RESULT_MOVIE, error) {
	url := API_URL + "/movie/" + fmt.Sprint(id) + "/similar?api_key=" + API_KEY + "&language=" + SHORT_LANGUAGE
	resp, err := http.Get(url)
	if err != nil {
		return []TMDB_SEARCH_RESULT_MOVIE{}, err
	}
	defer resp.Body.Close()
	var result TMDB_SEARCH_MOVIE
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Results, nil
}
func GetFullEpisodes(id int, numbers []int) ([]TMDB_FULL_SEASON, error) {
	var seasons []TMDB_FULL_SEASON
	fetchSeason := func(id int, season int, channel chan TMDB_FULL_SEASON, wg *sync.WaitGroup) {
		defer wg.Done()
		url := API_URL + "/tv/" + fmt.Sprint(id) + "/season/" + fmt.Sprint(season) + "?api_key=" + API_KEY + "&language=" + SHORT_LANGUAGE
		resp, err := http.Get(url)
		if err != nil {
			channel <- TMDB_FULL_SEASON{}
		}
		defer resp.Body.Close()
		var result TMDB_FULL_SEASON
		json.NewDecoder(resp.Body).Decode(&result)
		channel <- result
	}
	channel := make(chan TMDB_FULL_SEASON, len(numbers))
	var wg sync.WaitGroup
	for _, s := range numbers {
		wg.Add(1)
		go fetchSeason(id, s, channel, &wg)
	}
	wg.Wait()
	for i := 1; i <= len(numbers); i++ {
		seasons = append(seasons, <-channel)
	}
	return seasons, nil
}
func GetImage(images []TMDB_IMAGE_ITEM, ratio []float64) (TMDB_IMAGE_ITEM, error) {
	for _, language := range TMDB_IMAGE_PREFERED_LANGUAGE {
		for _, image := range images {
			for _, r := range ratio {
				if image.AspectRatio >= r && image.Iso_639_1 == language {
					return image, nil
				}
			}
		}
	}
	for _, image := range images {
		for _, r := range ratio {
			if image.AspectRatio >= r {
				return image, nil
			}
		}
	}
	return TMDB_IMAGE_ITEM{}, errors.New("no image found")
}

func GetVideo(videos []TMDB_VIDEO_ITEM) (TMDB_VIDEO_ITEM, error) {
	for _, video := range videos {
		if video.Type == "Trailer" && video.Site == "YouTube" {
			return video, nil
		}
	}
	// fmt.Println("No trailer found", len(videos))
	for _, video := range videos {
		if video.Site == "YouTube" {
			return video, nil
		}
	}
	return TMDB_VIDEO_ITEM{}, errors.New("no video found")
}

var cacheTmdb map[string]*TMDB_SEARCH_MOVIE = make(map[string]*TMDB_SEARCH_MOVIE)

func Get_tmdb_discover_movie(release_gte string, order string, release_lte string, watch_region string, withGenre []int, withKeywords []int, withOriginCountry string, withOriginalLanguage string, runtime_gte int32, runtime_lte int32, withWatchProviders []int) (*TMDB_SEARCH_MOVIE, error) {
	// https://api.themoviedb.org/3/watch/providers/movie?watch_region=FR&api_key=
	url := API_URL + "/discover/movie?api_key=" + API_KEY + "&language=" + SHORT_LANGUAGE
	if release_gte != "" {
		url += "&release_date.gte=" + release_gte
	}
	if release_lte != "" {
		url += "&release_date.lte=" + release_lte
	}
	if watch_region != "" {
		url += "&watch_region=" + watch_region
	}
	if order != "" {
		url += "&sort_by=" + order
	}

	if len(withGenre) > 0 {
		url += "&with_genres=" + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(withGenre)), ","), "[]")
	}
	if len(withKeywords) > 0 {
		url += "&with_keywords=" + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(withKeywords)), ","), "[]")
	}
	if withOriginCountry != "" {
		url += "&with_origin_country=" + withOriginCountry
	}
	if withOriginalLanguage != "" {
		url += "&with_original_language=" + withOriginalLanguage
	}
	if runtime_gte != -1 {
		url += "&with_runtime.gte=" + strconv.Itoa(int(runtime_gte))
	}
	if runtime_lte != -1 {
		url += "&with_runtime.lte=" + strconv.Itoa(int(runtime_lte))
	}
	if len(withWatchProviders) > 0 {
		url += "&with_watch_providers=" + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(withWatchProviders)), ","), "[]")
	}
	if val, ok := cacheTmdb[url]; ok {
		fmt.Println("Cache hit")
		return val, nil
	}
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	var result TMDB_SEARCH_MOVIE
	json.NewDecoder(resp.Body).Decode(&result)
	cacheTmdb[url] = &result
	return &result, nil
}
