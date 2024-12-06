package engine

import (
	"time"

	"gorm.io/gorm"
)

type Share struct {
	gorm.Model
	ID       uint      `gorm:"unique;not null,primary_key"`
	OWNER    User      `gorm:"foreignKey:OWNER_ID"`
	OWNER_ID uint      `gorm:"not null"`
	EXPIRE   time.Time `gorm:"not null"`
	FILE     FILE      `gorm:"foreignKey:FILE_ID"`
	FILE_ID  uint      `gorm:"not null"`
}

func (share *Share) GetOwner() *User {
	return &share.OWNER
}
func GetShareById(id int) *Share {
	var share Share
	if err := db.Where("id = ?", id).First(&share).Error; err != nil {
		return nil
	}
	return &share
}
func (share *Share) GetFile() *FILE {
	return &share.FILE
}
