package image

import (
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func HandlePosterController(source_type string, source_id string, target_image string, quality string, db *gorm.DB) ([]byte, error) {

	if slices.Contains([]string{"low", "high"}, quality) == false {
		return nil, engine.ErrorInvalidQuality
	}
	if source_type == engine.Tv {
		tv, err := engine.Get_tv_via_provider(source_id, false, func() *gorm.DB { return db })
		if err != nil {
			return nil, err
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
			return nil, engine.ErrorInvalidImage
		}
		if err != nil {
			return nil, err
		}
		return data, nil
	} else if source_type == engine.Movie {
		movie, err := engine.Get_movie_via_provider(source_id, false, func() *gorm.DB { return db })
		if err != nil {
			return nil, err
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
			return nil, engine.ErrorInvalidImage
		}
		if err != nil {
			return nil, err
		}
		return data, nil
	} else {
		return nil, engine.ErrorInvalidMediaType
	}

}

func HandlePoster(ctx *gin.Context, db *gorm.DB) {
	_, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	source_type := ctx.Query("type")
	source_id := ctx.Query("id")
	target_image := ctx.Query("target")
	quality := ctx.Query("quality")
	if data, err := HandlePosterController(source_type, source_id, target_image, quality, db); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	} else {
		ctx.Data(200, "image/png", data)
	}
}
