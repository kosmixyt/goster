package engine

var availableQualitys []string = []string{"1080p", "720p", "480p", "360p", "240p", "144p", "4k", "8k"}

type MediaQuality struct {
	ID                    uint `gorm:"primaryKey"`
	User                  *User
	UserID                uint
	MaxGoS                int64
	MinGoS                int64
	TargetGoS             int64
	Quality               string
	Title                 string
	MediaQualityProfileID uint
	MediaQualityProfile   *MediaQualityProfile
}
type MediaQualityProfile struct {
	ID            uint `gorm:"primaryKey"`
	User          *User
	UserID        uint
	MediaQualitys []MediaQuality `gorm:"foreignKey:MediaQualityProfileID"`
	Name          string
	IsMainOfUser  bool
}
