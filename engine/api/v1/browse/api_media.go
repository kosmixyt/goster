package browse

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func Browse(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	data, err := BrowseController(db, &user, ctx.Query("genre"), ctx.Query("type"), ctx.Query("provider"))
	if err == nil {
		ctx.JSON(200, gin.H{"elements": data})
		return
	}
	ctx.JSON(400, gin.H{"error": err.Error()})
}
func BrowseController(db *gorm.DB, user *engine.User, genre string, itype string, provider string) ([]engine.SKINNY_RENDER, error) {
	if genre != "" {
		var idgenre int
		_, err := fmt.Sscanf(genre, "%d", &idgenre)
		if err != nil {
			return nil, err
		}
		var Genre engine.GENRE
		if err := db.Where("id = ?", idgenre).Preload("TVS").Preload("TVS.GENRE").Preload("MOVIES").Preload("MOVIES.GENRE").Preload("MOVIES.WATCHING", "user_id = ?", user.ID).First(&Genre).Error; err != nil {
			return nil, err
		}
		elements := []engine.SKINNY_RENDER{}
		if itype == "movie" || itype == "" {
			for _, movie := range Genre.MOVIES {
				elements = append(elements, movie.Skinny(movie.GetWatching()))
			}
		}
		if itype == "tv" || itype == "" {
			for _, tv := range Genre.TVS {
				elements = append(elements, tv.Skinny(tv.GetWatching()))
			}
		}
		return elements, nil
	}
	if itype == "movie" {
		var movies []engine.MOVIE
		if err := db.Preload("GENRE").Preload("PROVIDERS").Preload("FILES").Preload("WATCHING", "user_id = ?", user.ID).Find(&movies).Error; err != nil {
			return nil, err
		}
		elements := []engine.SKINNY_RENDER{}
		for _, movie := range movies {
			elements = append(elements, movie.Skinny(movie.GetWatching()))
		}
		return elements, nil
	}
	if itype == "tv" {
		var tvs []engine.TV
		if err := db.Preload("GENRE").Preload("PROVIDERS").Preload("SEASON").Preload("SEASON.EPISODES").Preload("SEASON.EPISODES.FILES").Find(&tvs).Error; err != nil {
			return nil, err
		}
		elements := []engine.SKINNY_RENDER{}
		for _, tv := range tvs {
			elements = append(elements, tv.Skinny(tv.GetWatching()))
		}
		return elements, nil
	}
	if itype == "provider" {
		var tvs []engine.TV
		var movies []engine.MOVIE
		if err := db.Preload("GENRE").Joins("INNER JOIN tv_providers ON tvs.id = tv_providers.tv_id").Joins("INNER JOIN prov_id_ers ON prov_id_ers.id = tv_providers.prov_id_er_id").Preload("PROVIDERS").Preload("SEASON").Preload("SEASON.EPISODES").Preload("SEASON.EPISODES.FILES").
			Order("tvs.created_at desc").
			Where("prov_id_ers.id = ?", provider).
			Find(&tvs).Error; err != nil {
			return nil, err
		}
		if err := db.Preload("GENRE").Joins("INNER JOIN movie_providers ON movies.id = movie_providers.movie_id").Joins("INNER JOIN prov_id_ers ON prov_id_ers.id = movie_providers.prov_id_er_id").Preload("PROVIDERS").Preload("FILES").
			Where("prov_id_ers.id = ?", provider).
			Order("movies.created_at desc").
			Find(&movies).Error; err != nil {
			return nil, err
		}
		elements := []engine.SKINNY_RENDER{}
		for _, tv := range tvs {
			elements = append(elements, tv.Skinny(tv.GetWatching()))
		}
		for _, movie := range movies {
			elements = append(elements, movie.Skinny(movie.GetWatching()))
		}
		return elements, nil
	}
	return nil, fmt.Errorf("invalid query")
}
