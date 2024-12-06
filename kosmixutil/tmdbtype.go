package kosmixutil

type METADATA_FULL_MOVIE struct {
	Adult                 bool
	Backdrop_path         string
	Belongs_to_collection interface{}
	Budget                int
	Genres                []GENRE
	Homepage              string
	ID                    int
	Imdb_id               string
	Rated                 string
	Original_language     string
	OriginCountry         []string `json:"origin_country"`
	Original_title        string
	Overview              string
	Popularity            float64
	Poster_path           string
	Production_companies  []Production_companies
	Production_countries  []Production_countries
	Release_date          string
	Revenue               int
	Runtime               string
	Director              string
	Awards                string
	BoxOffice             string
	Writer                string
	Spoken_languages      []LANGUAGE
	Status                string
	Tagline               string
	Title                 string
	Vote_average          float64
	Vote_count            int
	Images                TMDB_IMAGE
	Videos                TMDB_VIDEOS `json:"videos"`
	// Alternative_titles    TMDB_ALTERNATIVE_TITLES_RESULTS `json:"alternative_titles"`
	WatchProviders TMDB_WATCH_PROVIDERS `json:"watch/providers"`
	External_ids   TMDB_EXTERNAL_IDS    `json:"external_ids"`
}
type TMDB_FULL_MOVIE struct {
	Adult                 bool
	Backdrop_path         string
	Belongs_to_collection interface{}
	Budget                int
	Genres                []GENRE
	Homepage              string
	ID                    int
	Imdb_id               string
	Original_language     string
	OriginCountry         []string `json:"origin_country"`
	Original_title        string
	Overview              string
	Popularity            float64
	Poster_path           string
	Production_companies  []Production_companies
	Production_countries  []Production_countries
	Release_date          string
	Revenue               int
	Runtime               int
	Spoken_languages      []LANGUAGE
	Status                string
	Tagline               string
	Title                 string
	Video                 bool
	Vote_average          float64
	Vote_count            int
	Images                TMDB_IMAGE
	Videos                TMDB_VIDEOS `json:"videos"`
	// Alternative_titles    TMDB_ALTERNATIVE_TITLES_RESULTS `json:"alternative_titles"`
	WatchProviders TMDB_WATCH_PROVIDERS `json:"watch/providers"`
	External_ids   TMDB_EXTERNAL_IDS    `json:"external_ids"`
}
type OMDB_FULL_ITEM struct {
	Title       string
	Year        string
	Rated       string
	Released    string
	Runtime     string
	Genre       string
	Director    string
	Writer      string
	Actors      string
	Plot        string
	Language    string
	Country     string
	Awards      string
	Poster      string
	Ratings     []OMDB_RATING
	Metascore   string
	ImdbRating  string `json:"imdbRating"`
	ImdbVotes   string `json:"imdbVotes"`
	ImdbID      string `json:"imdbID"`
	Type        string
	DVD         string
	BoxOffice   string
	Production  string
	Website     string
	TotalSeason string `json:"totalSeasons"`
}
type OMDB_RATING struct {
	Source string
	Value  string
}
type TMDB_EXTERNAL_IDS struct {
	Imdb_id      string `json:"imdb_id"`
	WikiData_id  string `json:"wikidata_id"`
	Facebook_id  string `json:"facebook_id"`
	Instagram_id string `json:"instagram_id"`
	Twitter_id   string `json:"twitter_id"`
	FreeBaseMid  string `json:"freebase_mid"`
	FreeBase_id  string `json:"freebase_id"`
	TvDb_id      int    `json:"tvdb_id"`
	TvRage_id    int    `json:"tvrage_id"`
}

type TMDB_ALTERNATIVE_TITLES_RESULTS struct {
	Titles []TMDB_ALTERNATIVE_TITLES `json:"titles"`
}
type TMDB_ALTERNATIVE_TITLES struct {
	Iso_3166_1 string `json:"iso_3166_1"`
	Title      string `json:"title"`
	Type       string `json:"type"`
}

type FRENCH_PROVIDER struct {
	Link string                `json:"link"`
	Rent []TMDB_WATCH_PROVIDER `json:"rent,omitempty"`
	Buy  []TMDB_WATCH_PROVIDER `json:"buy,omitempty"`
}

type TMDB_WATCH_PROVIDERS struct {
	Results struct {
		FR FRENCH_PROVIDER `json:"FR"`
	} `json:"results"`
}
type TMDB_WATCH_PROVIDER struct {
	Logo_path        string `json:"logo_path"`
	Provider_id      int    `json:"provider_id"`
	Provider_name    string `json:"provider_name"`
	Display_priority int    `json:"display_priority"`
}
type PROVIDER struct {
	Provider_name string
	Provider_url  string
	Provider_id   int
}
type TMDB_VIDEOS struct {
	Results []TMDB_VIDEO_ITEM `json:"results"`
}
type TMDB_VIDEO_ITEM struct {
	ID         string `json:"id"`
	Iso_639_1  string `json:"iso_639_1"`
	Iso_3166_1 string `json:"iso_3166_1"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Site       string `json:"site"`
	Size       int    `json:"size"`
	Type       string `json:"type"`
	Official   bool   `json:"official"`
	Published  bool   `json:"published"`
}
type TMDB_IMAGE struct {
	Backdrops []TMDB_IMAGE_ITEM `json:"backdrops"`
	Posters   []TMDB_IMAGE_ITEM `json:"posters"`
	Logo      []TMDB_IMAGE_ITEM `json:"logos"`
}
type TMDB_IMAGE_ITEM struct {
	AspectRatio float64 `json:"aspect_ratio"`
	FilePath    string  `json:"file_path"`
	Height      int     `json:"height"`
	Iso_639_1   string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int     `json:"vote_count"`
	Width       int     `json:"width"`
}

type METADATA_FULL_TV struct {
	Adult                bool
	Rated                string
	Backdrop_path        string
	Created_by           []TMDB_CREATED_BY
	Runtime              string
	First_air_date       string
	Genres               []GENRE
	Homepage             string
	ID                   int
	In_production        bool
	Languages            []string
	Director             string
	Writer               string
	Awards               string
	Last_air_date        string
	Last_episode_to_air  TMDB_LAST_EPISODE
	Name                 string
	Next_episode_to_air  interface{}
	Networks             []TMDB_NETWORK
	Number_of_episodes   int
	Number_of_seasons    int
	Origin_country       []string
	Original_language    string
	Original_name        string
	Overview             string
	Popularity           float64
	Poster_path          string
	Production_companies []Production_companies
	Production_countries []Production_countries
	Seasons              []TMDB_SEASON
	Spoken_languages     []LANGUAGE
	Status               string
	Tagline              string
	Type                 string
	Vote_average         float64
	Vote_count           int
	Video                TMDB_VIDEOS `json:"videos"`
	Images               TMDB_IMAGE
	WatchProviders       TMDB_WATCH_PROVIDERS `json:"watch/providers"`
	External_ids         TMDB_EXTERNAL_IDS    `json:"external_ids"`
}

type TMDB_FULL_SERIE struct {
	Adult                bool
	Backdrop_path        string
	Created_by           []TMDB_CREATED_BY
	Episode_run_time     []int
	First_air_date       string
	Genres               []GENRE
	Homepage             string
	ID                   int
	In_production        bool
	Languages            []string
	Last_air_date        string
	Last_episode_to_air  TMDB_LAST_EPISODE
	Name                 string
	Next_episode_to_air  interface{}
	Networks             []TMDB_NETWORK
	Number_of_episodes   int
	Number_of_seasons    int
	Origin_country       []string
	Original_language    string
	Original_name        string
	Overview             string
	Popularity           float64
	Poster_path          string
	Production_companies []Production_companies
	Production_countries []Production_countries
	Seasons              []TMDB_SEASON
	Spoken_languages     []LANGUAGE
	Status               string
	Tagline              string
	Type                 string
	Vote_average         float64
	Vote_count           int
	Video                TMDB_VIDEOS `json:"videos"`
	Images               TMDB_IMAGE
	WatchProviders       TMDB_WATCH_PROVIDERS `json:"watch/providers"`
	External_ids         TMDB_EXTERNAL_IDS    `json:"external_ids"`
}

type TMDB_SEASON struct {
	Air_date      string
	Episode_count int
	ID            int
	Name          string
	Overview      string
	Poster_path   string
	Season_number int
}
type TMDB_NETWORK struct {
	Name           string
	ID             int
	Logo_path      string
	Origin_country string
}

type TMDB_LAST_EPISODE struct {
	Id              int
	Name            string
	Overview        string
	Vote_average    float64
	Vote_count      int
	Air_date        string
	Episode_number  int
	Episode_type    string
	Production_code string
	Runtime         int
	Season_number   int
	Show_id         int
	Still_path      string
}
type TMDB_CREATED_BY struct {
	ID           int
	Credit_id    string
	Name         string
	Gender       int
	Profile_path string
}

type TMDB_FULL_SEASON struct {
	ID            int                 `json:"id"`
	Air_date      string              `json:"air_date"`
	Episodes      []TMDB_FULL_EPISODE `json:"episodes"`
	Name          string              `json:"name"`
	Overview      string              `json:"overview"`
	Poster_path   string              `json:"poster_path"`
	Season_number int                 `json:"season_number"`
	Vote_average  float64             `json:"vote_average"`
}
type TMDB_FULL_EPISODE struct {
	Air_date        string
	Episode_number  int
	ID              int
	Name            string
	Overview        string
	Production_code string
	Runtime         int
	Season_number   int
	Show_id         int
	Still_path      string
	Vote_average    float64
	Vote_count      int
	Crew            []TMDB_CREW
	Guest_stars     []TMDB_GUEST_STARS
}
type TMDB_CREW struct {
	Job                 string
	Department          string
	Credit_id           string
	Adult               bool
	Gender              int
	Know_for_department string
	Name                string
	Original_name       string
	Popularity          float64
	Profile_path        string
}
type TMDB_GUEST_STARS struct {
	Character            string
	Credit_id            string
	Order                int
	Adult                bool
	Gender               int
	ID                   int
	Known_for_department string
	Name                 string
	Original_name        string
	Popularity           float64
	Profile_path         string
}
type TMDB_SEARCH_SERIE struct {
	Page          int
	Results       []TMDB_SEARCH_RESULT_SERIE
	Total_pages   int
	Total_results int
}

type TMDB_SEARCH_RESULT_SERIE struct {
	Adult             bool
	Backdrop_path     string
	Genre_ids         []int
	ID                int
	Origin_country    []string
	Original_language string
	Original_name     string
	Overview          string
	Popularity        float64
	Poster_path       string
	First_air_date    string
	Name              string
	Vote_average      float64
	Vote_count        int
}

type GENRE struct {
	ID   uint
	Name string
}
type Production_companies struct {
	ID             int
	Logo_path      string
	Name           string
	Origin_country string
}
type Production_countries struct {
	Iso_3166_1   string
	Name         string
	English_name string
}
type LANGUAGE struct {
	Iso_639_1 string
	Name      string
}

type TMDB_SEARCH_MOVIE struct {
	Page          int
	Results       []TMDB_SEARCH_RESULT_MOVIE
	Total_pages   int
	Total_results int
}

type TMDB_SEARCH_RESULT_MOVIE struct {
	Adult             bool
	Backdrop_path     string
	Genre_ids         []int
	ID                int
	Original_language string
	Original_title    string
	Overview          string
	Popularity        float64
	Poster_path       string
	Release_date      string
	Title             string
	Video             bool
	Vote_average      float64
	Vote_count        int
}
type TMDB_MULTI_SEARCH struct {
	Page          int
	Results       []TMDB_MULTI_SEARCH_RESULT
	Total_pages   int
	Total_results int
}

type TMDB_MULTI_SEARCH_RESULT struct {
	Adult             bool
	Backdrop_path     string
	ID                int
	Title             string
	Name              string
	Original_language string
	Original_title    string
	Overview          string
	Poster_path       string
	Media_type        string
	Genre_ids         []int
	Popularity        float64
	Release_date      string
	Video             bool
	Vote_average      float64
	Vote_count        int
	Origin_country    []string
}
