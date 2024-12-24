package torrents

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func AvailableTorrentController(user *engine.User, db *gorm.DB, id string, itype string, season_id string) ([]TorrentItemRender, error) {
	var items []*engine.Torrent_File
	start := time.Now()
	if itype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id, true, user.RenderTvPreloads)
		if err != nil {
			return nil, err
		}
		season_id, err := strconv.Atoi(season_id)
		if err != nil {
			return nil, err
		}
		season := tvDbItem.GetExistantSeasonById(uint(season_id))
		if season.HasFile() {
			return nil, fmt.Errorf("season already downloaded")
		}
		preferedOrderedProvider, err := engine.FindBestTorrentFor(tvDbItem, nil, season, nil, 1, engine.GetMaxSize(engine.Movie))
		if err != nil {
			return nil, err
		}
		items = preferedOrderedProvider
	} else if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, user.RenderMoviePreloads)
		if err != nil {
			return nil, err
		}
		if movie.HasFile(nil) {
			return nil, fmt.Errorf("movie already downloaded")
		}
		preferedOrderedProvider, err := engine.FindBestTorrentFor(nil, movie, nil, nil, 1, engine.GetMaxSize(engine.Movie))
		if err != nil {
			return nil, err
		}
		items = preferedOrderedProvider
	} else {
		return nil, fmt.Errorf("type not found")
	}
	var wg sync.WaitGroup
	v := make(chan *TorrentItemRender, len(items))
	for _, item := range items {
		wg.Add(1)
		go MapTorrentItem(item, &wg, v, true)
	}
	wg.Wait()
	close(v)
	elapsed := time.Since(start)
	fmt.Println("Elapsed", elapsed)
	var res []TorrentItemRender = make([]TorrentItemRender, 0)
	for i := 0; i < len(items); i++ {
		item := <-v
		if item != nil {
			item.Flags = append(make([]string, 0), kosmixutil.GetCodec(item.Name), kosmixutil.GetQuality(item.Name), kosmixutil.GetSource(item.Name))
			res = append(res, *item)
		}
	}
	return res, nil
}

func AvailableTorrents(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	itype := ctx.Query("type")
	id := ctx.Query("id")
	season_id := ctx.Query("season")
	if data, err := AvailableTorrentController(&user, db, id, itype, season_id); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
	} else {
		ctx.JSON(200, data)
	}
}
func AvailableTorrentsWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"type", "id", "season"}, request.Options)
	if data, err := AvailableTorrentController(&user, db, keys["type"], keys["id"], keys["season"]); err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
	} else {
		kosmixutil.SendWebsocketResponse(conn, data, nil, request.RequestUuid)
	}
}
