package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func GetQualitys(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	qualitys, err := GetQualitysController(&user, db)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, qualitys)
}
func GetQualitysController(user *engine.User, db *gorm.DB) (*[]engine.QUALITY, error) {
	if !user.ADMIN {
		return nil, errors.New("unauthorized")
	}
	return &engine.QUALITYS, nil
}

func PostQuality(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var newQuality engine.QUALITY
	if err := c.BindJSON(&newQuality); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = PostQualityController(&user, &newQuality)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "quality added"})
}

func PostQualityController(user *engine.User, quality *engine.QUALITY) error {
	if !user.ADMIN {
		errors.New("unauthorized")
	}
	if quality.Width <= 0 || quality.Resolution <= 0 {
		return errors.New("invalid quality")
	}
	for _, q := range engine.QUALITYS {
		if q.Resolution == quality.Resolution {
			return errors.New("quality already exists")
		}
	}
	engine.QUALITYS = append(engine.QUALITYS, *quality)
	return nil
}
func DeleteQuality(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	res, err := strconv.Atoi(c.PostForm("resolution"))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = DeleteQualityController(&user, res)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "quality deleted"})
}

func DeleteQualityController(user *engine.User, Resolution int) error {
	if !user.ADMIN {
		return errors.New("unauthorized")
	}
	for i, q := range engine.QUALITYS {
		if q.Resolution == Resolution {
			engine.QUALITYS = append(engine.QUALITYS[:i], engine.QUALITYS[i+1:]...)
			return nil
		}
	}
	return errors.New("quality not found")
}

type TranscoderInfo struct {
	ID      uint                 `json:"id"`
	QUALITY string               `json:"quality"`
	SKINNY  engine.SKINNY_RENDER `json:"skinny"`
	Ip      string               `json:"ip"`
	Browser string               `json:"browser"`
}

func GetTranscoders(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	transcoders, err := GetTranscodersController(&user, db)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, transcoders)
}
func GetTranscodersController(user *engine.User, db *gorm.DB) (*[]TranscoderInfo, error) {
	if !user.ADMIN {
		return nil, errors.New("unauthorized")
	}
	var transcoders []TranscoderInfo = make([]TranscoderInfo, 0)
	for _, t := range engine.Transcoders {
		var transcoder TranscoderInfo
		transcoder.ID = t.OWNER_ID
		transcoder.QUALITY = t.CURRENT_QUALITY.Name
		file, ok := t.Source.(*engine.FILE)
		if ok {
			transcoder.SKINNY = file.SkinnyRender(user)
		} else {
			iptv_channel, ok := t.Source.(*engine.IptvChannel)
			if ok {
				transcoder.SKINNY = iptv_channel.Skinny()
			} else {
				transcoder.SKINNY = engine.SKINNY_RENDER{}
			}
		}
		transcoder.Ip = "127.0.0.1"
		transcoder.Browser = "Brave "
		transcoders = append(transcoders, transcoder)
	}
	return &transcoders, nil
}

func KillTranscoder(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	uuid := (c.Query("uuid"))
	err = KillTranscoderController(&user, uuid)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "transcoder killed"})
}

func KillTranscoderController(user *engine.User, uuid string) error {
	if !user.ADMIN {
		return errors.New("unauthorized")
	}
	for _, t := range engine.Transcoders {
		if t.UUID == uuid {
			t.Destroy("killed by admin")
			return nil
		}
	}
	return errors.New("transcoder not found")
}
