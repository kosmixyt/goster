package trailer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	youtube "github.com/kkdai/youtube/v2"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

var client = youtube.Client{}

func HandleTrailerRequest(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": err.Error()})
		return
	}
	if err := TrailerController(&user, ctx.Query("type"), ctx.Query("id"), ctx); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
	}
}
func TrailerController(user *engine.User, itype string, itemId string, ctx *gin.Context) error {
	var youtube_url string
	outpu_t_name := filepath.Join(engine.TRAILER_OUTPUT_PATH, itype+"_"+itemId+".mp4")
	if itype == engine.Tv {
		tvDbItem, err := engine.Get_tv_via_provider(itemId, false, user.RenderTvPreloads)
		if err != nil {
			return err
		}
		youtube_url = tvDbItem.TRAILER_URL
		return nil
	} else if itype == engine.Movie {
		movie, err := engine.Get_movie_via_provider(itemId, false, user.RenderMoviePreloads)
		if err != nil {
			return err
		}
		fmt.Println("Movie", movie.TRAILER_URL)
		youtube_url = movie.TRAILER_URL
	} else {
		return errors.New("Bad type")
	}
	if _, err := os.Stat(outpu_t_name); err == nil {
		ctx.File(outpu_t_name)
		return nil
	}
	fmt.Println("Downloading trailer", youtube_url)
	video, err := client.GetVideo(youtube_url)
	if err != nil {
		return err
	}
	format := video.Formats.Select(func(f youtube.Format) bool {
		return f.AudioQuality != "" && f.FPS != 0
	})
	if len(format) == 0 {
		return errors.New("No valid format found")
	}
	// 30mb
	if format[0].ContentLength > 300_000_00 {
		return errors.New("File too big")
	}
	stream, size, err := client.GetStream(video, &format[0])
	if err != nil {
		return err
	}
	file, err := os.Create(outpu_t_name)
	if err != nil {
		return err
	}
	defer file.Close()
	ctx.Header("Content-Length", strconv.FormatInt(size, 10))
	ctx.Header("Content-Disposition", "attachment; filename=trailer.mp4")
	writers := io.MultiWriter(ctx.Writer, file)
	io.Copy(writers, stream)
	return nil
}
