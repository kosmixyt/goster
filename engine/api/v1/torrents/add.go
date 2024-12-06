package torrents

import (
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TorrentAdd(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.CAN_UPLOAD {
		ctx.JSON(401, gin.H{"error": "not allowed to download"})
		return
	}
	var dl_item *engine.Torrent_File
	var pname = ctx.PostForm("addMethod")
	mediaType := ctx.PostForm("mediaType")
	task := user.CreateTask("add torrent to media", func() error { return errors.New("uncancellable task") })
	if pname == "manual" {
		formFile, err := ctx.FormFile("file")
		if err != nil || formFile == nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if formFile.Size > 100000000 {
			ctx.JSON(400, gin.H{"error": "file too big"})
			return
		}
		f, err := formFile.Open()
		if err != nil {
			ctx.JSON(500, gin.H{"error": "could not open file"})
			return
		}
		buffer, err := io.ReadAll(f)
		if err != nil || len(buffer) == 0 || len(buffer) > 100000000 {
			ctx.JSON(500, gin.H{"error": "could not read file"})
			return
		}
		dl_item = &engine.Torrent_File{
			UUID:      uuid.New().String(),
			NAME:      formFile.Filename + "-manual-",
			LINK:      "---null-link-manual",
			SEED:      1,
			LEECH:     0,
			PROVIDER:  "manual",
			SIZE_str:  "manual",
			MOVIE_ID:  nil,
			FetchData: "---null-fetchdata-manual",
		}
		db.Save(dl_item)
		dl_item.SetAsManual(buffer)

	} else if pname == "search" {
		// ctx.JSON(400, gin.H{"error": "not implemented"})
		// return
		for i, j := range ItemsTorrents {
			if j.LINK == ctx.PostForm("torrentId") {
				final := ItemsTorrents[i]
				dl_item = final
				fmt.Println("found")
				break
			}
		}
		if dl_item == nil {
			ctx.JSON(400, gin.H{"error": "torrent not found"})
			return
		}

	} else {
		ctx.JSON(400, gin.H{"error": "invalid addMethod"})
		return
	}

	if mediaType == engine.Movie {
		if movieDbItem, err := engine.Get_movie_via_provider(ctx.PostForm("mediauuid"), true, func() *gorm.DB { return db.Preload("FILES") }); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		} else {
			dl_item.MOVIE_ID = &movieDbItem.ID
			db.Save(&dl_item)
			if err := engine.AssignTorrentToMedia(db, &user, movieDbItem, nil, nil, dl_item, task); err != nil {
				ctx.JSON(500, gin.H{"error": task.SetAsError(err).(error).Error()})
				return
			}
		}
		ctx.JSON(200, gin.H{"status": "download added"})
		return
	}
	if mediaType == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(ctx.PostForm("mediauuid"), true, func() *gorm.DB {
			return db.Preload("SEASON").Preload("SEASON.EPISODES").Preload("SEASON.EPISODES.FILES")
		})
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var season_index int
		if _, err := fmt.Sscanf(ctx.PostForm("season_index"), "%d", &season_index); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid season_number" + err.Error()})
			return
		}
		if season_index >= len(tvDbItem.SEASON) {
			ctx.JSON(400, gin.H{"error": "invalid season_number no season matching"})
			return
		}
		season := tvDbItem.SEASON[season_index]
		if season.HasFile() {
			ctx.JSON(400, gin.H{"error": "season already has file"})
			return
		}
		dl_item.SEASON_ID = &season.ID

		dl_item.TV_ID = &tvDbItem.ID
		db.Save(&dl_item)
		fmt.Println(dl_item.MOVIE_ID)
		fmt.Println(dl_item.TV_ID)
		fmt.Println("---id of season---")
		if err := engine.AssignTorrentToMedia(db, &user, nil, season, tvDbItem, dl_item, task); err != nil {
			task.SetAsError(err)
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, gin.H{"status": "download added"})
		return
	}
	ctx.JSON(400, gin.H{"error": "invalid mediaType"})
}
