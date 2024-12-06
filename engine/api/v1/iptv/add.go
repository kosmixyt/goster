package iptv

import (
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AddIptv(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}

	reader, err := engine.GetIptvFileFromUrl(ctx.Query("url"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid url"})
		return
	}
	full, err := io.ReadAll(reader)
	if err != nil {
		full = nil
		ctx.JSON(400, gin.H{"error": "invalid url"})
		return
	}
	asStr := string(full)
	fmt.Println("Got Full", len(asStr))
	if err = engine.TestTextIptv(asStr); err != nil {
		fmt.Println("Test Failed")
		fmt.Println(err.Error())
		ctx.Data(400, "text/plain", []byte(err.Error()))
		return
	}
	reader = io.NopCloser(strings.NewReader(asStr))
	iptv, err := user.AddIptv(reader, 1)
	fmt.Println("Got Iptv")
	iptv.Init(&engine.Fid)
	engine.AppendIptv(iptv)
	ctx.JSON(200, gin.H{"success": "iptv added"})
}
