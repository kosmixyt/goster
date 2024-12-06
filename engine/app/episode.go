package engine

import (
	"fmt"

	"gorm.io/gorm"
)

type EPISODE struct {
	gorm.Model
	ID                       uint       `gorm:"unique;not null,primary_key"` // use
	DESCRIPTION              string     `gorm:"not null"`
	NAME                     string     `gorm:"not null"`
	FILES                    []FILE     `gorm:"foreignKey:EPISODE_ID"`
	SEASON_ID                uint       `gorm:"not null"`
	SEASON                   *SEASON    `gorm:"foreignKey:SEASON_ID"`
	NUMBER                   int        `gorm:"not null"`
	STILL_IMAGE_PATH         string     `gorm:"not null"`
	STILL_IMAGE_STORAGE_TYPE int        `gorm:"not null"`
	WATCHING                 []WATCHING `gorm:"foreignKey:EPISODE_ID"`
}

func (e *EPISODE) LoadSeason() {
	db.Model(&e).Association("SEASON").Find(&e.SEASON)
}

func (EPISODE) Get(id uint, preload []string) *EPISODE {
	var episode EPISODE
	for _, p := range preload {
		db.Preload(p)
	}
	if db.Where("id = ?", id).First(&episode).Error != nil {
		return nil
	}
	return &episode
}

func (e *EPISODE) GetNumberAsString(withC bool) string {
	c := ""
	if e.NUMBER < 10 && withC {
		c = "0"
	}
	return c + string(e.NUMBER)
}
func (e *EPISODE) HasFile(file *FILE) bool {
	if len(e.FILES) == 0 {
		fmt.Println("[WARN] No file found for episode", e.ID)
	}
	if file != nil {
		for _, f := range e.FILES {
			if f.ID == file.ID {
				return true
			}
		}
		return false
	}
	return len(e.FILES) > 0
}

func (e *EPISODE) ToFile(season *SEASON) []FileItem {
	files := []FileItem{}
	for _, file := range e.FILES {
		appnt := FileItem{
			ID:            file.ID,
			FILENAME:      file.FILENAME,
			SIZE:          file.SIZE,
			DOWNLOAD_URL:  fmt.Sprintf(Config.Web.PublicUrl+"/download?type=tv&id=db@%d&fileId=%d", season.TV_ID, file.ID),
			TRANSCODE_URL: fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=tv&id=db@%d&fileId=%d", season.TV_ID, file.ID),
		}
		if file.WATCHING != nil {
			if len(file.WATCHING) > 0 {
				appnt.CURRENT = file.WATCHING[0].CURRENT
			}
		}
		files = append(files, appnt)
	}
	return files
}
