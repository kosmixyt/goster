package admin

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func AdminInfo(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	data, err := GetAdminInfoController(&user, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, data)
}

var startTime = time.Now()

type AdminInfoData struct {
	Goroutines       int             `json:"goroutines"`
	TotalMemory      uint64          `json:"totalmemory"`
	CurrentTranscode int             `json:"currentTranscode"`
	Movies           int64           `json:"movies"`
	Series           int64           `json:"series"`
	Episodes         int64           `json:"episodes"`
	Files            int64           `json:"files"`
	Users            int64           `json:"users"`
	Port             string          `json:"port"`
	CPU              string          `json:"cpu"`
	GoVersion        string          `json:"goversion"`
	GPU              string          `json:"gpu"`
	Uptime           string          `json:"uptime"`
	RAM              uint64          `json:"ram"`
	Tasks            []AdminTaskData `json:"tasks"`
}
type AdminTaskData struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
	Name     string `json:"name"`
	Started  string `json:"started"`
	Status   string `json:"status"`
	Finished string `json:"finished"`
}

func GetAdminInfoController(user *engine.User, db *gorm.DB) (*AdminInfoData, error) {
	if !user.ADMIN {
		return nil, engine.ErrorIsNotAdmin
	}

	totalGoroutines := runtime.NumGoroutine()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	ss := engine.SystemInfo
	var movieCount, serieCount, episodeCount, fileCount int64
	db.Model(&engine.MOVIE{}).Count(&movieCount)
	db.Model(&engine.TV{}).Count(&serieCount)
	db.Model(&engine.EPISODE{}).Count(&episodeCount)
	db.Model(&engine.FILE{}).Count(&fileCount)

	data := AdminInfoData{
		Goroutines:       totalGoroutines,
		TotalMemory:      mem.Alloc,
		CurrentTranscode: len(engine.Transcoders),
		Movies:           movieCount,
		Series:           serieCount,
		Episodes:         episodeCount,
		Files:            fileCount,
		CPU:              ss.CPU.Vendor,
		GoVersion:        runtime.Version(),
		GPU:              ss.System.Model,
		Uptime:           time.Since(startTime).String(),
		Port:             engine.Config.Web.PublicPort,
		Tasks:            []AdminTaskData{},
	}
	var tasks []engine.Task
	db.Preload("User").Limit(10).Order("started desc").Find(&tasks)
	for _, task := range tasks {
		element := AdminTaskData{
			ID:     int(task.ID),
			Name:   task.Name,
			Status: task.Status,
		}
		if task.User != nil {
			element.UserName = task.User.NAME
		} else {
			element.UserName = "deleted user"
		}

		if task.Started != nil {
			element.Started = task.Started.Format("2006-01-02 15:04:05")
		}
		if task.Finished != nil {
			element.Finished = task.Finished.Format("2006-01-02 15:04:05")
		}
		data.Tasks = append(data.Tasks, element)
	}
	memo := 0
	for _, m := range ss.MemLayout {
		memo += int(m.Size)
	}
	data.RAM = uint64(memo)
	return &data, nil
}
