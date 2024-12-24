package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func GetTmdb(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	metadata, err := GetTmdbController(&user)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, metadata)
}

func SetTmdbController(user *engine.User, db *gorm.DB, metadata kosmixutil.MetadataBase) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	err := kosmixutil.VerifyMetadataBase(metadata)
	if err != nil {
		return err
	}
	engine.Config.Metadata.Omdb = metadata.Omdb
	engine.Config.Metadata.Tmdb = metadata.Tmdb
	engine.Config.Metadata.TmdbIso3166 = metadata.TmdbIso3166
	engine.Config.Metadata.TmdbLang = metadata.TmdbLang
	engine.Config.Metadata.TmdbImgLang = metadata.TmdbImgLang
	engine.NewConfig.Metadata = engine.Config.Metadata
	kosmixutil.InitKeys(engine.Config.Metadata.Tmdb, engine.Config.Metadata.Omdb, engine.Config.Metadata.TmdbImgLang, engine.Config.Metadata.TmdbLang)
	engine.ReWriteConfig()
	return nil
}

func GetTmdbController(user *engine.User) (kosmixutil.MetadataBase, error) {
	if !user.ADMIN {
		return kosmixutil.MetadataBase{}, engine.ErrorIsNotAdmin
	}
	return kosmixutil.MetadataBase{
		Tmdb:        engine.Config.Metadata.Tmdb,
		TmdbIso3166: engine.Config.Metadata.TmdbIso3166,
		Omdb:        engine.Config.Metadata.Omdb,
		TmdbLang:    engine.Config.Metadata.TmdbLang,
		TmdbImgLang: engine.Config.Metadata.TmdbImgLang,
	}, nil
}
func SetTmdb(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var metadata kosmixutil.MetadataBase
	err = ctx.ShouldBindJSON(&metadata)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = SetTmdbController(&user, db, metadata)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}
