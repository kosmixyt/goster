package metadata

import (
	"fmt"

	"github.com/gorilla/websocket"

	"sort"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func GetUnAssignedMediasController(user *engine.User, db *gorm.DB) (*DraggerData, error) {
	if !user.CAN_EDIT {
		return nil, fmt.Errorf("forbidden")
	}
	var movies []engine.MOVIE
	var tvs []engine.TV
	db.Preload("SEASON.EPISODES.FILES").Find(&tvs)
	db.Preload("FILES").Find(&movies)
	fmt.Println("movies", len(movies))
	moviesres := make([]MovieMetadata, len(movies))
	fmt.Println("len movieres", len(moviesres))
	for n, movie := range movies {
		m := MovieMetadata{
			ID:      movie.ID,
			NAME:    movie.NAME,
			TMDB_ID: movie.TMDB_ID,
			FILES:   make([]FILEMetadata, len(movie.FILES)),
		}
		if m.TMDB_ID == -1 {
			m.TMDB_ID = 0
		}
		for i, file := range movie.FILES {
			f := FILEMetadata{
				ID:   file.ID,
				NAME: file.FILENAME,
				PATH: engine.Joins(file.ROOT_PATH, file.SUB_PATH),
				SIZE: file.SIZE,
			}
			m.FILES[i] = f
		}
		// moviesres = append(moviesres, m)
		moviesres[n] = m
	}
	var tvsres []TvMetadata = make([]TvMetadata, len(tvs))
	for j, tv := range tvs {
		t := TvMetadata{
			ID:   tv.ID,
			NAME: tv.NAME, TMDB_ID: tv.TMDB_ID, SEASONS: make([]SeasonMetadata, len(tv.SEASON))}
		if t.TMDB_ID == -1 {
			t.TMDB_ID = 0
		}
		sort.Slice(tv.SEASON, func(i, j int) bool {
			return tv.SEASON[i].NUMBER < tv.SEASON[j].NUMBER
		})
		for a, season := range tv.SEASON {
			s := SeasonMetadata{
				ID:       season.ID,
				NAME:     season.NAME,
				NUMBER:   int(season.NUMBER),
				Episodes: make([]EpisodeMetadata, len(season.EPISODES)),
			}
			sort.Slice(season.EPISODES, func(i, j int) bool {
				return season.EPISODES[i].NUMBER < season.EPISODES[j].NUMBER
			})
			for b, episode := range season.EPISODES {
				e := EpisodeMetadata{
					ID:     episode.ID,
					NAME:   episode.NAME,
					NUMBER: int(episode.NUMBER),
					FILES:  make([]FILEMetadata, len(episode.FILES)),
				}
				for c, file := range episode.FILES {
					f := FILEMetadata{ID: (file.ID), NAME: file.FILENAME, PATH: engine.Joins(file.ROOT_PATH, file.SUB_PATH), SIZE: (file.SIZE)}
					e.FILES[c] = f
				}
				s.Episodes[b] = e
			}
			t.SEASONS[a] = s
		}
		tvsres[j] = t

	}
	var orphans []engine.FILE
	db.Where("tv_id IS NULL AND movie_id IS NULL").Find(&orphans)
	var orphansres []FILEMetadata = make([]FILEMetadata, len(orphans))
	for i, file := range orphans {
		orphansres[i] = FILEMetadata{ID: (file.ID), NAME: file.FILENAME, PATH: engine.Joins(file.ROOT_PATH, file.SUB_PATH), SIZE: (file.SIZE)}
	}
	return &DraggerData{Movies: moviesres, Tvs: tvsres, Orphans: orphansres}, nil

}

func GetUnAssignedMedias(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	data, err := GetUnAssignedMediasController(&user, db)
	if err != nil {
		ctx.JSON(403, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, data)
}

func GetUnAssignedMediasWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("not logged in"), request.RequestUuid)
		return
	}
	data, err := GetUnAssignedMediasController(&user, db)
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("error"), request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, data, nil, request.RequestUuid)
}
