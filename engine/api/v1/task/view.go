package task

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func GetTask(db *gorm.DB, ctx *gin.Context) {
	user, err := engine.GetUser(db, ctx, []string{"Tasks"})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	taskid, err := strconv.Atoi(ctx.Query("taskid"))
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid task id"})
		return
	}
	task := user.GetTask(taskid)
	if task == nil {
		ctx.JSON(404, gin.H{"error": "task not found"})
		return
	}
	kosmixutil.SendEvent(ctx, "log", string(task.GetLogs()))
	<-ctx.Writer.CloseNotify()

}
