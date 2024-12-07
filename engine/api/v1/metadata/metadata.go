package metadata

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

type MovieMetadata struct {
	ID      uint           `json:"id"`
	NAME    string         `json:"name"`
	TMDB_ID int            `json:"tmdb_id"`
	FILES   []FILEMetadata `json:"files"`
}
type FILEMetadata struct {
	ID   uint   `json:"id"`
	NAME string `json:"name"`
	PATH string `json:"path"`
	SIZE int64  `json:"size"`
}
type TvMetadata struct {
	ID      uint             `json:"id"`
	NAME    string           `json:"name"`
	TMDB_ID int              `json:"tmdb_id"`
	SEASONS []SeasonMetadata `json:"seasons"`
}
type SeasonMetadata struct {
	ID       uint              `json:"id"`
	NAME     string            `json:"name"`
	NUMBER   int               `json:"number"`
	Episodes []EpisodeMetadata `json:"episodes"`
}
type EpisodeMetadata struct {
	ID     uint           `json:"id"`
	NAME   string         `json:"name"`
	NUMBER int            `json:"number"`
	FILES  []FILEMetadata `json:"files"`
}

func GetUnAssignedMedias(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.CAN_EDIT {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
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
				// s.Episodes = append(s.Episodes, e)
				s.Episodes[b] = e
			}
			// t.SEASONS = append(t.SEASONS, s)
			t.SEASONS[a] = s
		}
		// tvsres = append(tvsres, t)
		tvsres[j] = t

	}
	var orphans []engine.FILE
	db.Where("tv_id IS NULL AND movie_id IS NULL").Find(&orphans)
	var orphansres []FILEMetadata = make([]FILEMetadata, len(orphans))
	for i, file := range orphans {
		orphansres[i] = FILEMetadata{ID: (file.ID), NAME: file.FILENAME, PATH: engine.Joins(file.ROOT_PATH, file.SUB_PATH), SIZE: (file.SIZE)}
	}
	ctx.JSON(200, gin.H{"movies": moviesres, "tvs": tvsres, "orphans": orphansres})

}

func GetDraggerData(ctx *gin.Context, db *gorm.DB) {}

func AssignFileToMedia(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.CAN_EDIT {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	fileId, err := strconv.Atoi(ctx.PostForm("fileid"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid fileid"})
		return
	}
	var file engine.FILE
	db.Where("id = ?", fileId).First(&file)
	if file.ID == 0 {
		ctx.JSON(404, gin.H{"error": "file not found"})
		return
	}
	ntype, id := ctx.PostForm("type"), ctx.PostForm("id")
	if ntype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id, true, user.RenderTvPreloads)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		season_id, err := strconv.Atoi(ctx.PostForm("season_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid season_id"})
			return
		}
		episode_id, err := strconv.Atoi(ctx.PostForm("episode_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid episode_id"})
			return
		}
		s := tvDbItem.GetExistantSeasonById(uint(season_id))
		if s == nil {
			ctx.JSON(400, gin.H{"error": "season not found"})
			return
		}
		e := s.GetExistantEpisodeById(uint(episode_id))
		if e == nil {
			ctx.JSON(400, gin.H{"error": "episode not found"})
			return
		}
		db.Updates(&engine.FILE{ID: file.ID, TV_ID: tvDbItem.ID, EPISODE_ID: e.ID, SEASON_ID: s.ID}).
			Update("movie_id", gorm.Expr("null")).
			Where("id = ?", file.ID)
	} else if ntype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, user.RenderMoviePreloads)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		db.
			Updates(&engine.FILE{ID: file.ID, MOVIE_ID: movie.ID}).
			Update("tv_id", gorm.Expr("null")).
			Update("episode_id", gorm.Expr("null")).
			Update("season_id", gorm.Expr("null")).
			Where("id = ?", file.ID)
	} else if ntype == "orphan" {
		db.
			Updates(&engine.FILE{ID: file.ID}).
			Update("tv_id", gorm.Expr("null")).
			Update("episode_id", gorm.Expr("null")).
			Update("season_id", gorm.Expr("null")).
			Update("movie_id", gorm.Expr("null")).
			Where("id = ?", file.ID)
	} else {
		ctx.JSON(400, gin.H{"error": "invalid type"})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})

}
func ClearMoviesWithNoMediaAndNoTmdbId(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	var movies []engine.MOVIE
	db.Preload("FILES").Where("tmdb_id = ?", -1).Find(&movies)
	deleted := 0
	for _, movie := range movies {
		if len(movie.FILES) == 0 {
			if tx := db.Delete(&movie); tx.Error != nil {
				ctx.JSON(500, gin.H{"error": tx.Error.Error()})
				return
			}
			deleted++
		}
	}
	ctx.JSON(200, gin.H{"message": "deleted " + strconv.Itoa(deleted) + " movies"})
}

// rename strategy
// func RenameAllFiles(ctx *gin.Context, db *gorm.DB) {
// 	user, err := engine.GetUser(db, ctx, []string{})
// 	if err != nil {
// 		ctx.JSON(401, gin.H{"error": "not logged in"})
// 		return
// 	}
// }
