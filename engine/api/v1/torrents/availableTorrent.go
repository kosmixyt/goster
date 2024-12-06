package torrents

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AvailableTorrent(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	itype := ctx.Query("type")
	id := ctx.Query("id")
	var items []*engine.Torrent_File
	start := time.Now()

	if itype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(id, true, user.RenderTvPreloads)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		season_id, err := strconv.Atoi(ctx.Query("season"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		season := tvDbItem.GetExistantSeasonById(uint(season_id))
		if season.HasFile() {
			ctx.JSON(400, gin.H{"error": "season already downloaded"})
			return
		}
		preferedOrderedProvider, err := engine.FindBestTorrentFor(tvDbItem, nil, season, nil, 1, engine.GetMaxSize(engine.Movie))
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		items = preferedOrderedProvider
	} else if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(id, true, user.RenderMoviePreloads)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if movie.HasFile(nil) {
			ctx.JSON(400, gin.H{"error": "movie already downloaded"})
			return
		}
		preferedOrderedProvider, err := engine.FindBestTorrentFor(nil, movie, nil, nil, 1, engine.GetMaxSize(engine.Movie))
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		items = preferedOrderedProvider
	} else {
		ctx.JSON(400, gin.H{"error": "no type"})
		return
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
			res = append(res, *item)
		}
	}
	ctx.JSON(200, gin.H{"torrents": res})
}
