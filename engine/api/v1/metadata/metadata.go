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

type DraggerData struct {
	Movies  []MovieMetadata `json:"movies"`
	Tvs     []TvMetadata    `json:"tvs"`
	Orphans []FILEMetadata  `json:"orphans"`
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

func AssignFileToMediaController(user *engine.User, db *gorm.DB, fileIdstr string, ntype string, id string, season_id_str string, episode_id_str string) error {
	if !user.CAN_EDIT {
		return fmt.Errorf("forbidden")
	}
	fileId, err := strconv.Atoi(fileIdstr)
	if err != nil {
		return fmt.Errorf("invalid fileid")
	}
	var file engine.FILE
	db.Where("id = ?", fileId).First(&file)
	if file.ID == 0 {
		return fmt.Errorf("file not found")
	}
	if ntype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id, true, user.RenderTvPreloads)
		if err != nil {
			return err
		}
		season_id, err := strconv.Atoi(season_id_str)
		if err != nil {
			return fmt.Errorf("invalid season_id")
		}
		episode_id, err := strconv.Atoi(episode_id_str)
		if err != nil {
			return fmt.Errorf("invalid episode_id")
		}
		s := tvDbItem.GetExistantSeasonById(uint(season_id))
		if s == nil {
			return fmt.Errorf("season not found")
		}
		e := s.GetExistantEpisodeById(uint(episode_id))
		if e == nil {
			return fmt.Errorf("episode not found")
		}
		db.Updates(&engine.FILE{ID: file.ID, TV_ID: tvDbItem.ID, EPISODE_ID: e.ID, SEASON_ID: s.ID}).
			Update("movie_id", gorm.Expr("null")).
			Where("id = ?", file.ID)
	} else if ntype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, user.RenderMoviePreloads)
		if err != nil {
			return err
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
		return fmt.Errorf("invalid type")
	}
	return nil
}
func AssignFileToMedia(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	err = AssignFileToMediaController(&user, db, ctx.PostForm("file_id"), ctx.PostForm("type"), ctx.PostForm("id"), ctx.PostForm("season_id"), ctx.PostForm("episode_id"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
	}
}
func ClearMoviesWithNoMediaAndNoTmdbIdController(db *gorm.DB, user *engine.User) (string, error) {
	if !user.ADMIN {
		return "", fmt.Errorf("forbidden")
	}
	var movies []engine.MOVIE
	db.Preload("FILES").Where("tmdb_id = ?", -1).Find(&movies)
	deleted := 0
	for _, movie := range movies {
		if len(movie.FILES) == 0 {
			if tx := db.Delete(&movie); tx.Error != nil {
				return "", tx.Error
			}
			deleted++
		}
	}
	return "deleted " + strconv.Itoa(deleted) + " movies", nil
}
func ClearMoviesWithNoMediaAndNoTmdbId(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	msg, err := ClearMoviesWithNoMediaAndNoTmdbIdController(db, &user)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": msg})
}

func BulkSerieMoveController(user *engine.User, db *gorm.DB, source_id_str string, target_id_str string) error {
	if !user.CAN_EDIT {
		return fmt.Errorf("forbidden")
	}
	source_id := source_id_str
	target_id := target_id_str
	fmt.Println("source_id", source_id, "target_id", target_id)
	source, err := engine.Get_tv_via_provider(source_id, true, user.RenderTvPreloads)
	if err != nil {
		return err
	}
	target, err := engine.Get_tv_via_provider(target_id, true, user.RenderTvPreloads)
	if err != nil {
		return err
	}
	if source.ID == target.ID {
		return fmt.Errorf("source and target are the same")
	}
	if err := source.MoveFiles(target); err != nil {
		return err
	}
	return nil
}
func BulkSerieMove(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	err = BulkSerieMoveController(&user, db, ctx.PostForm("source_id"), ctx.PostForm("target_id"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}
