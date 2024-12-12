package dlrequest

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func NewDownloadRequest(db *gorm.DB, ctx *gin.Context) {
	user, err := engine.GetUser(db, ctx, []string{"Requests"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err, req := NewDownloadRequestController(ctx.Query("max_size"), &user, ctx.Query("season_id"), ctx.Query("id"), ctx.Query("type")); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	} else {
		ctx.JSON(200, gin.H{"status": "success", "id": req.ID})
	}

}
func NewDownloadRequestWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{"Requests"})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	vals := kosmixutil.GetStringKeys([]string{"max_size", "season_id", "id", "type"}, request.Options)
	if err, req := NewDownloadRequestController(vals["max_size"], &user, vals["season_id"], vals["id"], vals["type"]); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
	} else {
		kosmixutil.SendWebsocketResponse(conn, gin.H{"status": "success", "id": req.ID}, nil, request.RequestUuid)
	}
}
func NewDownloadRequestController(max_size_str string, user *engine.User, season_id_str string, id_str string, itype string) (error, *engine.DownloadRequest) {
	max_size, err := strconv.ParseInt(max_size_str, 10, 64)
	if max_size == 0 || err != nil {
		return errors.New("max_size is required"), nil
	}
	var req *engine.DownloadRequest
	if itype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id_str, true, user.RenderTvPreloads)
		if err != nil {
			return err, nil
		}
		season_id, err := strconv.ParseUint(season_id_str, 10, 64)
		if err != nil {
			return err, nil
		}
		season := tvDbItem.GetExistantSeasonById(uint(season_id))
		if season == nil {
			return errors.New("season not found"), nil
		}
		if max_size > engine.GetMaxSize(engine.Tv) {
			return errors.New("max_size is too large"), nil
		}
		if user.GetTvRequest(tvDbItem.ID, uint(season_id)) != nil {
			return errors.New("request already exists"), nil
		}
		if season.HasFile() {
			return errors.New("season already has a file"), nil
		}
		treq := user.NewRequestDownload(max_size, nil, season, tvDbItem)
		req = treq
	} else if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id_str, true, user.RenderMoviePreloads)
		if err != nil {
			return err, nil
		}
		if max_size > engine.GetMaxSize(engine.Movie) {
			return errors.New("max_size is too large"), nil
		}
		if user.GetMovieRequest(movie.ID) != nil {
			return errors.New("request already exists"), nil
		}
		if movie.HasFile(nil) {
			return errors.New("movie already has a file"), nil
		}
		treq := user.NewRequestDownload(max_size, movie, nil, nil)
		req = treq
	}
	return nil, req
}
