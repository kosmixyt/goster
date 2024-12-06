package share

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func DeleteShare(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{"SHARES"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if err := DeleteShareController(db, ctx.Query("id"), user); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "ok"})
}
func DeleteShareController(db *gorm.DB, id string, user engine.User) error {
	intid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	sh := user.GetShareId(intid)
	if sh == nil {
		return errors.New("share not found")
	}
	db.Delete(sh)
	return nil
}
