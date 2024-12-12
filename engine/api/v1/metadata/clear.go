package metadata

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
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

type DraggerData struct {
	Movies  []MovieMetadata `json:"movies"`
	Tvs     []TvMetadata    `json:"tvs"`
	Orphans []FILEMetadata  `json:"orphans"`
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
func ClearMoviesWithNoMediaAndNoTmdbIdWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("not logged in"), request.RequestUuid)
		return
	}
	msg, err := ClearMoviesWithNoMediaAndNoTmdbIdController(db, &user)
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("error"), request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"message": msg}, nil, request.RequestUuid)
}
