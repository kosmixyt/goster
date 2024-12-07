package image

import (
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func HandlePoster(ctx *gin.Context, db *gorm.DB) {
	source_type := ctx.Query("type")
	source_id := ctx.Query("id")
	target_image := ctx.Query("image")
	quality := ctx.Query("quality")
	if slices.Contains([]string{"low", "high"}, quality) == false {
		ctx.JSON(400, gin.H{
			"error": "Invalid quality",
		})
		return
	}
	if source_type == engine.Tv {
		tv, err := engine.Get_tv_via_provider(source_id, false, func() *gorm.DB { return db })
		if err != nil {
			ctx.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		var data []byte
		switch target_image {
		case "poster":
			rtmp, etmp := tv.GetPoster(quality)
			data = rtmp
			err = etmp
		case "backdrop":
			rtmp, etmp := tv.GetBackdrop(quality)
			data = rtmp
			err = etmp
		case "logo":
			rtmp, etmp := tv.GetLogo(quality)
			data = rtmp
			err = etmp
		default:
			ctx.JSON(400, gin.H{
				"error": "Invalid image",
			})
			return
		}
		if err != nil {
			ctx.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		ctx.Data(200, "image/jpeg", data)
	} else if source_type == engine.Movie {
		movie, err := engine.Get_movie_via_provider(source_id, false, func() *gorm.DB { return db })
		if err != nil {
			ctx.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		var data []byte
		switch target_image {
		case "poster":
			rtmp, etmp := movie.GetPoster(quality)
			data = rtmp
			err = etmp
		case "backdrop":
			rtmp, etmp := movie.GetBackdrop(quality)
			data = rtmp
			err = etmp
		case "logo":
			rtmp, etmp := movie.GetLogo(quality)
			data = rtmp
			err = etmp

		default:
			ctx.JSON(400, gin.H{
				"error": "Invalid image",
			})
			return
		}
		if err != nil {
			ctx.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		ctx.Data(200, "image/jpeg", data)
	} else {
		ctx.JSON(400, gin.H{
			"error": "Invalid type",
		})
		return
	}

}
