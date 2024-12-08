package engine

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

var oneWeekAgo = time.Now().AddDate(0, 0, -30).Format("2006-01-02")

func GetBestGenre(watching *[]WATCHING) []*GenreClassement {
	var BestGenres = make([]*GenreClassement, 0)
	InsertGenre := func(genre GENRE, ItemType string, render *ITEM_INF) {
		for _, bg := range BestGenres {
			if bg.Genre.ID == genre.ID {
				if render.TMDB_ID != -1 {
					bg.Items = append(bg.Items, render)
				}
				bg.Count++
				return
			}
		}
		BestGenres = append(BestGenres, &GenreClassement{
			Items:    make([]*ITEM_INF, 0),
			Genre:    genre,
			Count:    1,
			ItemType: ItemType,
		})
	}
	for _, w := range *watching {
		if w.MOVIE != nil {
			for _, g := range w.MOVIE.GENRE {
				InsertGenre(g, "movie", &ITEM_INF{
					TMDB_ID: w.MOVIE.TMDB_ID,
					TYPE:    Movie,
					NAME:    w.MOVIE.NAME,
				},
				)
			}
		}
		if w.TV != nil {
			for _, g := range w.TV.GENRE {
				InsertGenre(g, "tv", &ITEM_INF{
					TMDB_ID: w.TV.TMDB_ID,
					TYPE:    Tv,
					NAME:    w.TV.NAME,
				})
			}
		}
	}
	sort.SliceStable(BestGenres, func(i, j int) bool {
		return BestGenres[i].Count > BestGenres[j].Count
	})
	if len(BestGenres) > 10 {
		BestGenres = BestGenres[:10]
	}
	return BestGenres
}

func GetGenreRecommendation(watching *[]WATCHING, db *gorm.DB, preloadMovie []string, preloadTv []string, user *User) ([]Line_Render, []*GenreClassement) {
	BestGenres := GetBestGenre(watching)
	BestGenreLineRender := func(wg *sync.WaitGroup, channel chan Line_Render, bg *GenreClassement, db *gorm.DB, user *User, preloadMovie []string, preloadTv []string) {
		defer wg.Done()
		movies, tvs, render := make([]MOVIE, 0), make([]TV, 0), make([]SKINNY_RENDER, 0)
		if bg.ItemType == "movie" {
			req := db.Table("movies").Joins("INNER JOIN movie_genres ON movies.id = movie_genres.movie_id").Where("movie_genres.genre_id = ?", bg.Genre.ID).Preload("WATCHING", "USER_ID = ? ", user.ID).Preload("PROVIDERS").Preload("WATCHLISTS", "id = ? ", user.ID).Limit(50)
			for _, p := range preloadMovie {
				req = req.Preload(p)
			}
			req.Find(&movies)
		} else {
			req := db.Table("tvs").Joins("INNER JOIN tv_genres ON tvs.id = tv_genres.tv_id").Where("tv_genres.genre_id = ?", bg.Genre.ID).Preload("WATCHING", "USER_ID = ? ", user.ID).Preload("PROVIDERS").Preload("WATCHLISTS", "id = ? ", user.ID).Limit(50)
			for _, p := range preloadTv {
				req = req.Preload(p)
			}
			req.Find(&tvs)

		}
		render = append(render, MapMovieSkinny(movies)...)
		render = append(render, MapTvSkinny(tvs)...)
		channel <- Line_Render{
			Data:  render,
			Title: bg.Genre.NAME,
			Type:  "items",
		}
	}
	var wg sync.WaitGroup
	mediane := len(BestGenres) / 2
	// on prend 50% des genres les plus regardés
	BestGenres = append(BestGenres[mediane:], BestGenres[:mediane]...)
	var renderers = make(chan Line_Render, len(BestGenres)*3)
	for _, bg := range BestGenres {
		wg.Add(1)
		go BestGenreLineRender(&wg, renderers, bg, db, user, preloadMovie, preloadTv)
	}
	fmt.Println("waiting for genre recommendations")
	wg.Wait()
	fmt.Println("done waiting for genre recommendations")
	close(renderers)
	var lines []Line_Render = make([]Line_Render, 0)
	for r := range renderers {
		if len(r.Data) > 10 && len(r.Data) > 6 {
			lines = append(lines, r)
		}
	}
	return lines, BestGenres

}

func GetProviderRender(db *gorm.DB, channel chan []Provider_Line, wg *sync.WaitGroup) {
	defer wg.Done()
	providers, items := make([]PROVIDER, 0), make([]PROVIDERItem, 0)
	db.Find(&providers)
	for _, p := range providers {
		items = append(items, ParseProviderItem([]PROVIDER{p})[0])
	}
	channel <- []Provider_Line{
		Provider_Line{
			Data:  items,
			Title: "Providers",
			Type:  "providers",
		},
	}
}

func GetProviderLineRender(db *gorm.DB, channel chan []Line_Render, wg *sync.WaitGroup, provider_name string, provider_id int) {
	fmt.Println("Date one week ago:", oneWeekAgo)
	defer wg.Done()
	results, err := kosmixutil.Get_tmdb_discover_movie("", "primary_release_date.desc", "", Config.Metadata.TmdbIso3166, []int{}, []int{}, "", "", -1, -1, []int{provider_id})
	if err != nil {
		fmt.Println("Error getting discover movie", err)
		return
	}
	var render = make([]SKINNY_RENDER, 0)
	for _, r := range results.Results {
		sk := TmdbSkinnyRender(&r, nil, nil)
		render = append(render, sk)
	}
	channel <- []Line_Render{
		Line_Render{
			Data:  render,
			Title: "New Movies On Provider " + provider_name,
			Type:  "items",
		},
	}
}

func GetRecomFromGenreOnProvider(db *gorm.DB, channel chan []Line_Render, wg *sync.WaitGroup, provider_id int, provider_name string, genre GENRE) {
	defer wg.Done()
	results, err := kosmixutil.Get_tmdb_discover_movie("", "popularity.desc", "", Config.Metadata.TmdbIso3166, []int{int(genre.ID)}, []int{}, "", "", -1, -1, []int{provider_id})
	if err != nil {
		fmt.Println("Error getting discover movie", err)
		return
	}
	var render = make([]SKINNY_RENDER, 0)
	for _, r := range results.Results {
		sk := TmdbSkinnyRender(&r, nil, nil)
		render = append(render, sk)
	}
	channel <- []Line_Render{
		Line_Render{
			Data:  render,
			Title: "New Movies On Provider " + provider_name + " in genre " + genre.NAME,
			Type:  "items",
		},
	}

}
