package watching

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func DeleteFromWatchingListController(user *engine.User, db *gorm.DB, elementType string, uuid string) error {
	if elementType != engine.Tv && elementType != engine.Movie {
		return fmt.Errorf("invalid type")
	}
	provider, id, err := engine.ParseIdProvider(uuid)
	if err != nil {
		return err
	}
	if provider != "db" {
		return fmt.Errorf("invalid provider")
	}
	field := elementType + "_id"
	if tx := db.Where("user_id = ? AND "+field+" = ?", user.ID, id).Delete(&engine.WATCHING{}); tx.Error != nil {
		return tx.Error
	}
	return nil
}

func DeleteFromWatchingList(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err := DeleteFromWatchingListController(&user, db, ctx.Query("type"), ctx.Query("id")); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
}
