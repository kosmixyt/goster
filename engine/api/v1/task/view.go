package task

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
func GetTaskWs(db *gorm.DB, request kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{"Tasks"})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("not logged in"), request.RequestUuid)
		return
	}
	key := kosmixutil.GetStringKeys([]string{"taskid"}, request.Options)
	taskid, err := strconv.Atoi(key["taskid"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("invalid task id"), request.RequestUuid)
		return
	}
	task := user.GetTask(taskid)
	if task == nil {
		kosmixutil.SendWebsocketResponse(conn, nil, errors.New("task not found"), request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, string(task.GetLogs()), nil, request.RequestUuid)
}
