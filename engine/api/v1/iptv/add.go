package iptv

import (
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AddIptvController(user *engine.User, url string) error {

	reader, err := engine.GetIptvFileFromUrl(url)
	if err != nil {
		return err
	}
	full, err := io.ReadAll(reader)
	if err != nil {
		full = nil
		return err
	}
	asStr := string(full)
	fmt.Println("Got Full", len(asStr))
	if err = engine.TestTextIptv(asStr); err != nil {
		fmt.Println("Test Failed")
		fmt.Println(err.Error())
		return err
	}
	reader = io.NopCloser(strings.NewReader(asStr))
	iptv, err := user.AddIptv(reader, 1)
	if err != nil {
		return err
	}
	fmt.Println("Got Iptv")
	iptv.Init(&engine.Fid)
	engine.AppendIptv(iptv)
	return nil
}
func AddIptv(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		return
	}
	if err := AddIptvController(&user, ctx.Query("url")); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
}
