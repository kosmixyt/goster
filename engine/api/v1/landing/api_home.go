package landing

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func Landing(db *gorm.DB, ctx *gin.Context) {
	start := time.Now()
	user, err := engine.GetUser(db, ctx, []string{"WATCH_LIST_MOVIES", "WATCH_LIST_TVS"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	data, err := LandingController(&user, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, data)
	fmt.Println("time", time.Since(start))
}
func LandingWesocket(db *gorm.DB, request *kosmixutil.WebsocketMessage, websocket *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{"WATCH_LIST_MOVIES", "WATCH_LIST_TVS"})
	if err != nil {
		fmt.Println("not logged in")
		kosmixutil.SendWebsocketResponse(websocket, nil, fmt.Errorf("not logged in"), request.RequestUuid)
		return
	}
	fmt.Println("user", user)
	data, err := LandingController(&user, db)
	fmt.Println("data", err)
	if err != nil {
		kosmixutil.SendWebsocketResponse(websocket, nil, fmt.Errorf("error"), request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(websocket, data, nil, request.RequestUuid)
}
func LandingController(user *engine.User, db *gorm.DB) (interface{}, error) {
	recents := engine.GetRecent(db, *user)
	var lines []engine.Line_Render = make([]engine.Line_Render, 0)
	WATCHINGS := user.GetReworkedWatching()
	lines = append(lines, engine.Line_Render{
		Title: "Watching",
		Data:  engine.MapWatching(WATCHINGS),
		Type:  "items",
	})
	lineOfGenre, BestGenres := engine.GetGenreRecommendation(&WATCHINGS, db, []string{"GENRE"}, []string{"GENRE"}, user)
	if len(BestGenres) > 10 {
		BestGenres = BestGenres[:10]
	}
	mv, tvs := user.GetWatchList()
	asSkinny := engine.MapMovieSkinny(mv)
	asSkinny = append(asSkinny, engine.MapTvSkinny(tvs)...)
	lines = append(lines,
		engine.Line_Render{
			Data:  asSkinny,
			Title: "Watchlist",
			Type:  "items",
		},
	)
	lines = append(lines, lineOfGenre...)
	countOfChannel := 2 + ((1 + len(BestGenres)) * len(engine.Config.Metadata.TmdbMovieWatchProviders))
	channel := make(chan []engine.Line_Render, countOfChannel)
	var wg sync.WaitGroup
	channelsProvider := make(chan []engine.Provider_Line, 1)
	wg.Add(1)
	go user.Most_Viewed(channel, &wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		mv, tvs := user.GetBestRated()
		asSkinny := engine.MapMovieSkinny(mv)
		asSkinny = append(asSkinny, engine.MapTvSkinny(tvs)...)
		channel <- []engine.Line_Render{
			engine.Line_Render{
				Data:  asSkinny,
				Title: "Best Rated",
				Type:  "items",
			},
		}
	}()

	wg.Add(1)
	go engine.GetProviderRender(db, channelsProvider, &wg)
	for name, providerId := range engine.Config.Metadata.TmdbMovieWatchProviders {
		wg.Add(1)
		go engine.GetProviderLineRender(db, channel, &wg, name, providerId)
		for _, bg := range BestGenres {
			wg.Add(1)
			go engine.GetRecomFromGenreOnProvider(db, channel, &wg, providerId, name, bg.Genre)
		}
	}
	wg.Wait()
	for i := 0; i < countOfChannel; i++ {
		lines = append(lines, <-channel...)
	}

	wg.Wait()
	for i, l := range lines {
		if len(l.Data) > 6 {
			lines[i].Data = l.Data[0 : len(l.Data)-(len(l.Data)%6)]
		}
	}
	return gin.H{
		"Recents":   recents,
		"Lines":     lines,
		"Providers": <-channelsProvider,
	}, nil

}
