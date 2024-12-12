package render

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func RenderItem(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	err, movie, tv := RenderItemController(ctx.Query("type"), ctx.Query("id"), &user, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if movie != nil {
		ctx.JSON(200, movie)
		return
	}
	if tv != nil {
		ctx.JSON(200, tv)
		return
	}
}
func RenderItemController(itype string, id string, user *engine.User, db *gorm.DB) (error, *engine.MovieItem, *engine.TVItem) {
	if itype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id, true, user.RenderTvPreloads)
		if err != nil {
			return err, nil, nil
		}
		go db.Model(&tvDbItem).Update("view", gorm.Expr("view + 1"))
		renderer := tvDbItem.Render(user)
		return nil, nil, &renderer
	} else if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, user.RenderMoviePreloads)
		if err != nil {
			return err, nil, nil
		}
		go db.Model(&movie).Update("view", gorm.Expr("view + 1"))
		renderer := movie.Render(user)
		return nil, &renderer, nil
	}
	return errors.New("bad type"), nil, nil
}
func RenderItemWs(db *gorm.DB, request kosmixutil.WebsocketMessage, websocket *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(websocket, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	vals := kosmixutil.GetStringKeys([]string{"type", "id"}, request.Options)
	err, movie, tv := RenderItemController(vals["type"], vals["id"], &user, db)
	if err != nil {
		kosmixutil.SendWebsocketResponse(websocket, nil, err, request.RequestUuid)
		return
	}
	if movie != nil {
		kosmixutil.SendWebsocketResponse(websocket, movie, nil, request.RequestUuid)
		return
	}
	if tv != nil {
		kosmixutil.SendWebsocketResponse(websocket, tv, nil, request.RequestUuid)
		return
	}
}
