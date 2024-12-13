package main

import (
	"fmt"

	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/engine/app/diffusion"
)

func main() {
	engine.LoadConfig()
	db := engine.Init()
	if db == nil {
		fmt.Println("test")
		return
	}
	fmt.Println("Init Finished Starting Web server")
	port := diffusion.WebServer(db, engine.Config.Web.PublicPort)
	diffusion.SetupWebsocketClient(db, port)
}
