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
	err = AssignFileToMediaController(&user, db, ctx.PostForm("fileid"), ctx.PostForm("type"), ctx.PostForm("id"), ctx.PostForm("season_id"), ctx.PostForm("episode_id"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
	}
}
func AssignFileToMediaWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, fmt.Errorf("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"file_id", "type", "id", "season_id", "episode_id"}, request.Options)
	err = AssignFileToMediaController(&user, db, keys["file_id"], keys["type"], keys["id"], keys["season_id"], keys["episode_id"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, gin.H{"message": "ok"}, nil, request.RequestUuid)
}
