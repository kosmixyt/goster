package diffusion

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"kosmix.fr/streaming/engine/api/v1/landing"
	"kosmix.fr/streaming/kosmixutil"
)

func SetupWebsocketClient(db *gorm.DB) {
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
