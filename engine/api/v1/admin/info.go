package admin

import (
	"runtime"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func AdminInfo(ctx *gin.Context, db *gorm.DB) {
	storages := engine.GetPathWithFreeSpaceStr()
	totalGoroutines := runtime.NumGoroutine()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	ss := engine.SystemInfo

	data := gin.H{
		"goroutines":    totalGoroutines,
		"totalmemory":   mem.Alloc,
		"cpus":          runtime.NumCPU(),
		"goversion":     runtime.Version(),
		"os":            runtime.GOOS,
		"paths":         storages,
		"download_path": engine.GetAvailableSizeInDownloadPath(),
		"users":         GetUserResume(db),
		"info": gin.H{
			"virtual":        ss.System.Virtual,
			"model":          ss.System.Model,
			"distro_version": ss.Os.Distro + " " + ss.Os.Release,
			"hostname":       ss.Os.Hostname,
			"physical_cores": ss.CPU.PhysicalCores,
			"logical_cores":  ss.CPU.Cores,
			"processor":      ss.CPU.Processors,
			"cpu_vendor":     ss.CPU.Vendor,
			// "dynamic":        kosmixutil.SubstribeDynamic,
			"uptime": kosmixutil.SubstribeDynamic.Time.Uptime,
			"load":   kosmixutil.SubstribeDynamic.CurrentLoad.AvgLoad,
		},
	}
	memo := 0
	for _, m := range ss.MemLayout {
		memo += int(m.Size)
	}
	data["info"].(gin.H)["memory"] = memo

	ctx.JSON(200, data)
}
func GetUserResume(db *gorm.DB) []UserResume {
	var users []engine.User
	db.Preload("TORRENTS").Preload("Requests").Find(&users)
	var userResumes []UserResume
	for _, user := range users {
		item := UserResume{
			Name:             user.NAME,
			TorrrentCount:    len(user.TORRENTS),
			TotalTorrentSize: 0,
			RealTorrentSize:  0,
			RequestCount:     len(user.Requests),
		}
		for _, torrent := range user.TORRENTS {
			item.TotalTorrentSize += torrent.Size
			to := engine.GetTorrent(torrent.ID)
			if to == nil {
				panic("torrent not found")
			}
			item.RealTorrentSize += to.Torrent.BytesCompleted()
		}
		item.RemainingSpace = user.ALLOWED_UPLOAD_SIZE - item.TotalTorrentSize
		userResumes = append(userResumes, item)
	}
	return userResumes
}

type UserResume struct {
	Name             string `json:"name"`
	TorrrentCount    int    `json:"torrent_count"`
	TotalTorrentSize int64  `json:"total_torrent_size"`
	RealTorrentSize  int64  `json:"real_torrent_size"`
	RequestCount     int    `json:"request_count"`
	RemainingSpace   int64  `json:"remaining_space"`
}
