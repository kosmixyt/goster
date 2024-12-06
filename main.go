package main

import (
	"fmt"

	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/engine/app/web"
)

func main() {

	engine.LoadConfig()
	db := engine.Init()
	if db == nil {
		fmt.Println("test")
		return
	}
	fmt.Println("Init Finished Starting Web server")
	web.WebServer(db, engine.Config.Web.PublicPort)
}
