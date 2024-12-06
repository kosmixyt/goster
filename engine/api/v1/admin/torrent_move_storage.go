package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

// func MoveTargetStorage(ctx *gin.Context, db *gorm.DB) {
// 	user, err := engine.GetUser(db, ctx, []string{})
// 	if err != nil {
// 		ctx.JSON(401, gin.H{"error": "not logged in"})
// 		return
// 	}
// 	if !user.ADMIN {
// 		ctx.JSON(401, gin.H{"error": "not admin"})
// 		return
// 	}
// 	split := engine.Config.Scan_paths
// 	targetPath := ctx.Query("target")
// 	if !slices.Contains(split, targetPath) {
// 		kosmixutil.SendEvent(ctx, "move", "error target path must be in scan paths")
// 		return
// 	}
// 	torrentId := ctx.Query("torrent_id")
// 	torrentint, err := strconv.Atoi(torrentId)
// 	if err != nil {
// 		kosmixutil.SendEvent(ctx, "move", "error invalid torrent_id")
// 		return
// 	}
// 	torrentItem := engine.GetTorrent(uint(torrentint))
// 	if torrentItem == nil {
// 		kosmixutil.SendEvent(ctx, "move", "error torrent not found")
// 		return
// 	}
// 	var item engine.Torrent
// 	if tx := db.Where("id = ?", torrentItem.DB_ID).First(&item); tx.Error != nil {
// 		kosmixutil.SendEvent(ctx, "move", "error torrent not found in db")
// 		return
// 	}
// 	Progress := func(m int64, t int64) {
// 		fmt.Println("Progress", m, t)
// 		kosmixutil.SendEvent(ctx, "progress", strconv.FormatFloat(float64(m)/float64(t), 'f', 2, 64))
// 	}
// 	success := make(chan error, 1)
// 	go engine.MoveTargetStorage(torrentItem.Torrent, &item, db, targetPath, &Progress, success)
// 	fmt.Println("Waiting for move to finish")
// 	err = <-success
// 	fmt.Println("Move finished")
// 	if err != nil {
// 		kosmixutil.SendEvent(ctx, "move", "error moving torrent"+err.Error())
// 	} else {
// 		kosmixutil.SendEvent(ctx, "move", "success")
// 	}
// }

func GetAvailablePaths(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(401, gin.H{"error": "not admin"})
		return
	}
	renders, err := engine.GetStorageRenders()
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, renders)

}
