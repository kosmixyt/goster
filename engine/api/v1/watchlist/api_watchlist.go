package watchlist

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func WatchListEndpoint(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{"WATCH_LIST_MOVIES", "WATCH_LIST_TVS"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err, list := WatchListController(ctx.Query("action"), ctx.Query("type"), ctx.Query("id"), &user,
		db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	} else {
		ctx.JSON(200, list)
	}

}
func DeleteFromWatchingListWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKeys([]string{"action", "type", "id"}, request.Options)
	if err, list := WatchListController(key["action"], key["type"], key["id"], &user, db); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	} else {
		kosmixutil.SendWebsocketResponse(conn, list, nil, request.RequestUuid)
	}
}

func WatchListController(action string, itype string, id string, user *engine.User, db *gorm.DB) (error, []engine.SKINNY_RENDER) {
	if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, func() *gorm.DB { return db })
		if err != nil {
			return err, nil
		}
		if action == "remove" {
			user.RemoveWatchListMovie(*movie)
		}
		if action == "add" {
			user.AddWatchListMovie(*movie)
		}
	} else if itype == engine.Tv {
		tv, err := engine.Get_tv_via_provider(id, true, func() *gorm.DB { return db })
		if err != nil {
			return err, nil
		}
		if action == "remove" {
			user.RemoveWatchListTv(*tv)
		}
		if action == "add" {
			user.AddWatchListTv(*tv)
		}
	}
	mv, tvs := user.GetWatchList()
	asSkinny := engine.MapMovieSkinny(mv)
	asSkinny = append(asSkinny, engine.MapTvSkinny(tvs)...)
	return nil, asSkinny
}
