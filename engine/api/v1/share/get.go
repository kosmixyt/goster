package share

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func GetShare(ctx *gin.Context, db *gorm.DB) {
	share, err := GetShareController(db, ctx.Query("id"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	f := share.GetFile()
	kosmixutil.ServerRangeRequest(ctx, f.SIZE, f.GetReader(), true, true)
}

func GetShareController(db *gorm.DB, id string) (*engine.Share, error) {
	var idint, err = strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	var share *engine.Share
	if err := db.Where("id = ?", idint).Preload("FILE").First(&share).Error; err != nil {
		return nil, err
	}
	if time.Now().After(share.EXPIRE) {
		fmt.Println("expired", time.Now().After(share.EXPIRE), share.EXPIRE, time.Now())
		db.Delete(share)
		return nil, errors.New("expired")
	}
	return share, nil
}
