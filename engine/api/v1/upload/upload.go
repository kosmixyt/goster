package upload

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

var tempsUploads = make(map[string]string)

func UploadFile(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	action := c.PostForm("action")
	if action == "start" {
		name := c.PostForm("name")
		size := c.PostForm("size")
		intsize, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid size"})
			return
		}

		if intsize < int64(1_000_000) {
			c.JSON(400, gin.H{"error": "file too small"})
			return
		}
		storer, root_path_of, err := engine.ParsePath(c.PostForm("path"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		outputType := c.PostForm("type")
		var movies *engine.MOVIE
		var episodes *engine.EPISODE
		if outputType == engine.Movie {
			movie, err := engine.Get_movie_via_provider(c.PostForm("uuid"), true, func() *gorm.DB { return db })
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			if movie.HasFile(nil) {
				c.JSON(400, gin.H{"error": "file already uploaded"})
				return
			}
			movies = movie
		} else if outputType == engine.Tv {
			episode_id, err := strconv.Atoi(c.PostForm("episode_id"))
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid episode number"})
				return
			}
			var episode *engine.EPISODE
			db.Where("id = ?", episode_id).First(&episode)
			if episode.ID == 0 {
				c.JSON(400, gin.H{"error": "episode not found"})
				return
			}
			episodes = episode
		} else {
			c.JSON(400, gin.H{"error": "invalid type"})
			return
		}
		upload, err := user.Upload(storer, root_path_of, name, intsize, movies, episodes)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": upload.ID})
	}
	if action == "upload" {
		upl, err := strconv.Atoi(c.PostForm("upload_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid upload id"})
			return
		}
		upload := user.GetUpload(uint(upl))
		if upload == nil {
			c.JSON(400, gin.H{"error": "upload not found"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		fh, err := file.Open()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		full, err := io.ReadAll(fh)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		fmt.Println("Wrote", len(full), "bytes")
		fmt.Println(float64(upload.CURRENT) / float64(upload.TOTAL) * 100)
		upload.Write(full)
		full = nil
		c.JSON(200, gin.H{"message": "ok"})
	}
}

// user, err := engine.GetUser(db, c, []string{})
// 	if err != nil {
// 		c.JSON(401, gin.H{"error": "not logged in"})
// 		return
// 	}
// 	if !user.CAN_ADD_FILES {
// 		c.JSON(401, gin.H{"error": "not allowed to upload"})
// 		return
// 	}
// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(400, gin.H{"error": err.Error()})
// 		return
// 	}
// 	if !user.CanUpload(file.Size) {
// 		c.JSON(400, gin.H{"error": "file too big"})
// 		return
// 	}
// 	if file.Size < int64(math.Pow10(6)) {
// 		c.JSON(400, gin.H{"error": "file too small"})
// 		return
// 	}
// 	// movie size max == episode size max
// 	if file.Size > engine.GetMaxSize(engine.Movie) {
// 		c.JSON(400, gin.H{"error": "file too big"})
// 		return
// 	}
// 	reader, err := file.Open()
// 	if err != nil {
// 		c.JSON(400, gin.H{"error": err.Error()})
// 		return
// 	}
// 	defer reader.Close()
// 	var movieDbItem *engine.MOVIE
// 	var tvDbItem *engine.TV
// 	var seasonDb *engine.SEASON
// 	var episodeDb *engine.EPISODE
// 	if !slices.Contains(engine.Config.Scan_paths, c.PostForm("path")) {
// 		c.JSON(400, gin.H{"error": "path not found"})
// 		return
// 	}
// 	outputpath := Joins(c.PostForm("path"), file.Filename)
// 	if c.PostForm("type") == engine.Movie {
// 		if t, err := engine.Get_movie_via_provider(c.PostForm("id"), true, func() *gorm.DB { return db.Preload("FILES") }); err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		} else {
// 			movieDbItem = t
// 		}
// 		if movieDbItem.HasFile(nil) {
// 			c.JSON(400, gin.H{"error": "file already uploaded"})
// 			return
// 		}
// 	} else if c.PostForm("type") == engine.Tv {
// 		t, err := engine.Get_tv_via_provider(c.PostForm("id"), true, func() *gorm.DB {
// 			return db.Preload("SEASON").Preload("SEASON.EPISODES")
// 		})
// 		var season_id, episode_id int
// 		if _, err := fmt.Sscanf(c.PostForm("season"), "%d", &season_id); err != nil {
// 			c.JSON(400, gin.H{"error": "invalid season number"})
// 			return
// 		}
// 		if _, err := fmt.Sscanf(c.PostForm("episode"), "%d", &episode_id); err != nil {
// 			c.JSON(400, gin.H{"error": "invalid episode number"})
// 			return
// 		}
// 		season := t.GetExistantSeasonById(uint(season_id))
// 		if season == nil {
// 			c.JSON(400, gin.H{"error": "season not found"})
// 			return
// 		}
// 		seasonDb = season
// 		episode := season.GetExistantEpisodeById(uint(episode_id))
// 		if episode == nil {
// 			c.JSON(400, gin.H{"error": "episode not found"})
// 			return
// 		}
// 		episodeDb = episode
// 		tvDbItem = t
// 		if err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		}
// 	} else {
// 		c.JSON(400, gin.H{"error": "type not found"})
// 		return
// 	}
// 	if _, err := os.Stat(outputpath); os.IsNotExist(err) {
// 		out, err := os.Create(outputpath)
// 		if err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		}
// 		defer out.Close()
// 		io.Copy(out, reader)
// 	} else {
// 		if err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		}
// 	}
// 	dbFile := engine.FILE{
// 		IS_MEDIA: true,
// 		FILENAME: file.Filename,
// 		PATH:     c.PostForm("path"),
// 		SIZE:     file.Size,
// 	}
// 	if c.PostForm("type") == engine.Movie {
// 		dbFile.MOVIE_ID = movieDbItem.ID
// 	}
// 	if c.PostForm("type") == engine.Tv {
// 		dbFile.TV_ID = tvDbItem.ID
// 		dbFile.SEASON_ID = seasonDb.ID
// 		dbFile.EPISODE_ID = episodeDb.ID
// 	}
// 	db.Save(&dbFile)
// 	user.Add_upload(file.Size)
// 	c.JSON(200, gin.H{"message": "ok"})
