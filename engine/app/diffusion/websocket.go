package diffusion

var uuidLength = 36
var binaryMessageTypeLength = 8

// func SetupWebsocketClient(db *gorm.DB, app *gin.Engine) {
// 	serverURL := "ws://localhost:4040"

// 	// Dial the WebSocket server
// tk:
// 	for {
// 		fmt.Println("Connecting to WebSocket server...")
// 		conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
// 		if err != nil {
// 			fmt.Println("Failed to connect to WebSocket server: %v", err)
// 		}
// 		defer conn.Close()
// 		fmt.Println("Connected to WebSocket server")
// 		interrupt := make(chan os.Signal, 1)
// 		signal.Notify(interrupt, os.Interrupt)

// 	_:
// 		for {
// 			_, msg, err := conn.ReadMessage()
// 			if err != nil {
// 				log.Printf("Error reading message: %v", err)
// 				continue tk
// 			}
// 			message, binaryMessage := ParsePacket(msg)
// 			if binaryMessage != nil {
// 				fmt.Println("Received binary message: ", binaryMessage)
// 				continue
// 			}
// 			log.Printf("Received message: %v", message)
// 			switch message.Type {
// 			case "home":
// 				landing.LandingWesocket(db, message, conn)
// 			case "render":
// 				render.RenderItemWs(db, message, conn)
// 			case "availableTorrent":
// 				torrents.AvailableTorrentsWs(db, message, conn)
// 			case "task":
// 				task.GetTaskWs(db, message, conn)
// 			case "me":
// 				me.HandleMeWs(db, message, conn)
// 			case "watchlist.get":
// 				watchlist.DeleteFromWatchingListWs(db, message, conn)
// 			case "image":
// 				image.HandlePosterWs(db, message, conn)
// 			case "search":
// 				search.MultiSearchWs(db, message, conn)
// 			case "browse":
// 				browse.BrowseWs(db, message, conn)
// 			case "iptv.ordered":
// 				iptv.OrderedIptvWs(db, message, conn)
// 			case "iptv.logo":
// 				iptv.LogoWs(db, message, conn)
// 			case "transcode.new":
// 				transcode.NewTranscoderWs(app, db, message, conn)
// 			case "transcode.segment":
// 				transcode.TranscodeSegmentWs(db, message, conn)
// 			case "transcode.subtitles":
// 				transcode.TranscodeSubtitleWs(db, message, conn)
// 			case "torrent.search":
// 				torrents.SearchTorrentsWs(db, message, conn)
// 			case "share.add":
// 				share.AddShareWs(db, message, conn)
// 			case "share.delete":
// 				share.DeleteShareWs(db, message, conn)
// 			default:
// 				fmt.Println("Unknown message type: %v", message.Type)
// 			}
// 		}
// 	}
// }

func GetStringKey(key string, opt interface{}) string {
	val, exist := opt.(map[string]interface{})[key].(string)
	if !exist {
		return ""
	}
	return val
}

// func ParsePacket(packet []byte) (*kosmixutil.WebsocketMessage, *kosmixutil.WebsocketBinaryMesage) {
// 	packet_type := packet[:4]
// 	fmt.Println("Packet type: ", (packet_type))
// 	if string(packet_type) == "json" {
// 		var message kosmixutil.WebsocketMessage
// 		err := json.Unmarshal(packet[4:], &message)
// 		if err != nil {
// 			panic(err)
// 		}
// 		return &message, nil
// 	}

// 	if string(packet_type) == "byte" {
// 		request_uuid := packet[4 : 4+uuidLength]
// 		return nil, &kosmixutil.WebsocketBinaryMesage{
// 			RequestUuid: string(request_uuid),
// 			Type:        string(packet[4+uuidLength : 4+uuidLength+binaryMessageTypeLength]),
// 			Data:        packet[4+uuidLength+binaryMessageTypeLength:],
// 		}
// 	}
// 	fmt.Println("Unknown packet type: ", string(packet_type))
// 	return nil, nil
// }
