package diffusion

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"kosmix.fr/streaming/engine/api/v1/browse"
	"kosmix.fr/streaming/engine/api/v1/image"
	"kosmix.fr/streaming/engine/api/v1/iptv"
	"kosmix.fr/streaming/engine/api/v1/landing"
	"kosmix.fr/streaming/engine/api/v1/me"
	"kosmix.fr/streaming/engine/api/v1/render"
	"kosmix.fr/streaming/engine/api/v1/search"
	"kosmix.fr/streaming/engine/api/v1/share"
	"kosmix.fr/streaming/engine/api/v1/task"
	"kosmix.fr/streaming/engine/api/v1/torrents"
	"kosmix.fr/streaming/engine/api/v1/transcode"
	"kosmix.fr/streaming/engine/api/v1/watchlist"
	"kosmix.fr/streaming/kosmixutil"
)

func SetupWebsocketClient(db *gorm.DB, app *gin.Engine) {
	serverURL := "ws://localhost:4040"

	// Dial the WebSocket server
tk:
	for {
		fmt.Println("Connecting to WebSocket server...")
		conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
		if err != nil {
			fmt.Println("Failed to connect to WebSocket server: %v", err)
		}
		defer conn.Close()
		fmt.Println("Connected to WebSocket server")
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)

	_:
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue tk
			}
			var message kosmixutil.WebsocketMessage
			err = json.Unmarshal(msg, &message)
			if err != nil {
				panic(err)
			}
			log.Printf("Received message: %v", message)
			switch message.Type {
			case "home":
				landing.LandingWesocket(db, message, conn)
			case "render":
				render.RenderItemWs(db, message, conn)
			case "availableTorrent":
				torrents.AvailableTorrentsWs(db, message, conn)
			case "task":
				task.GetTaskWs(db, message, conn)
			case "me":
				me.HandleMeWs(db, message, conn)
			case "watchlist.get":
				watchlist.DeleteFromWatchingListWs(db, message, conn)
			case "image":
				image.HandlePosterWs(db, message, conn)
			case "search":
				search.MultiSearchWs(db, message, conn)
			case "browse":
				browse.BrowseWs(db, message, conn)
			case "iptv.ordered":
				iptv.OrderedIptvWs(db, message, conn)
			case "iptv.logo":
				iptv.LogoWs(db, message, conn)
			case "transcode.new":
				transcode.NewTranscoderWs(app, db, message, conn)
			case "transcode.segment":
				transcode.TranscodeSegmentWs(db, message, conn)
			case "transcode.subtitles":
				transcode.TranscodeSubtitleWs(db, message, conn)
			case "torrent.search":
				torrents.SearchTorrentsWs(db, message, conn)
			case "share.add":
				share.AddShareWs(db, message, conn)
			case "share.delete":
				share.DeleteShareWs(db, message, conn)
			}
		}
	}
}

func GetStringKey(key string, opt interface{}) string {
	val, exist := opt.(map[string]interface{})[key].(string)
	if !exist {
		return ""
	}
	return val
}
