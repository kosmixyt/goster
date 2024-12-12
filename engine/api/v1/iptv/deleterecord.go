package iptv

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func RemoveRecordController(record_id string, db *gorm.DB) error {
	recordId, err := strconv.ParseInt(record_id, 10, 64)
	if err != nil {
		return err
	}
	var record engine.Record
	if db.Where("id = ?", recordId).First(&record).Error != nil {
		return engine.ErrorRecordNotFound
	}
	if err := db.Delete(&record).Error; err != nil {
		return err
	}
	return nil
}

func RemoveRecord(ctx *gin.Context, db *gorm.DB) {
	_, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err := RemoveRecordController(ctx.Param("record_id"), db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

}
