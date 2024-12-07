package search

import (
	"errors"
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func MultiSearch(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if data, err := SearchController(db, ctx.Query("query"), ctx.Query("type"), &user); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	} else {
		ctx.JSON(200, data)
	}
}
func SearchController(db *gorm.DB, q string, specificType string, user *engine.User) ([]engine.SKINNY_RENDER, error) {
	var queryengine = []string{engine.Tv, engine.Movie}
	if specificType != "" {
		if !slices.Contains(queryengine, specificType) {
			return nil, errors.New("invalid type")
		}
		queryengine = []string{specificType}
	}
	var data, serr = kosmixutil.MultiSearch(q)
	if serr != nil {
		return nil, errors.New("error while searching")
	}
	fromTMDB := data.Results
	elements := []engine.SKINNY_RENDER{}
	for _, movie := range fromTMDB {
		found := false
		for _, item := range elements {
			if item.NAME == movie.Title && item.DESCRIPTION == movie.Overview {
				found = true
			}
		}
		if movie.Media_type == "tv" && slices.Contains(queryengine, engine.Tv) {
			if !found {
				elements = append(elements, engine.TmdbSkinnyRender(nil, nil, &movie))
			}
		} else {
			if movie.Media_type == "movie" && slices.Contains(queryengine, engine.Movie) {
				if !found {
					elements = append(elements, engine.TmdbSkinnyRender(nil, nil, &movie))
				}
			}
		}
	}
	if slices.Contains(queryengine, engine.Movie) {
		var Movies []engine.MOVIE
		user.SkinnyMoviePreloads().Where("name LIKE ?", "%"+q+"%").Find(&Movies)
		for _, movie := range Movies {
			elements = append(elements, movie.Skinny(movie.GetWatching()))
		}
	}
	if slices.Contains(queryengine, engine.Tv) {
		var TV []engine.TV
		user.SkinnyTvPreloads().Where("name LIKE ?", "%"+q+"%").Find(&TV)
		for _, tv := range TV {
			elements = append(elements, tv.Skinny(tv.GetWatching()))
		}
	}
	return elements, nil
}
