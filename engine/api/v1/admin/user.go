package admin

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

type UserData struct {
	Id                  uint   `json:"id"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Token               string `json:"token"`
	Admin               bool   `json:"admin"`
	CanDownload         bool   `json:"can_download"`
	CanConvert          bool   `json:"can_convert"`
	CanAddFiles         bool   `json:"can_add_files"`
	CanUpload           bool   `json:"can_upload"`
	CanDelete           bool   `json:"can_delete"`
	CanEdit             bool   `json:"can_edit"`
	CanTranscode        bool   `json:"can_transcode"`
	MaxTranscoding      int    `json:"max_transcoding"`
	AllowedUploadNumber int    `json:"allowed_upload_number"`
	AllowedUploadSize   int    `json:"allowed_upload_size"`
}

func GetUsers(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	users, err := GetUserController(&user, db)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, users)
}

func GetUserController(user *engine.User, db *gorm.DB) (*[]UserData, error) {
	if !user.ADMIN {
		return nil, errors.New("not admin")
	}
	var users []engine.User
	db.Find(&users)
	var usersData []UserData
	for _, user := range users {
		usersData = append(usersData, UserData{
			Id:                  user.ID,
			Name:                user.NAME,
			Email:               user.EMAIL,
			Token:               user.TOKEN,
			Admin:               user.ADMIN,
			CanDownload:         user.CAN_DOWNLOAD,
			CanConvert:          user.CAN_CONVERT,
			CanAddFiles:         user.CAN_ADD_FILES,
			CanUpload:           user.CAN_UPLOAD,
			CanDelete:           user.CAN_DELETE,
			CanEdit:             user.CAN_EDIT,
			CanTranscode:        user.CAN_TRANSCODE,
			MaxTranscoding:      user.MAX_TRANSCODING,
			AllowedUploadNumber: int(user.ALLOWED_UPLOAD_NUMBER),
			AllowedUploadSize:   int(user.ALLOWED_UPLOAD_SIZE),
		})
	}
	return &usersData, nil
}
func UpdateUser(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var userData UserData
	if err := c.BindJSON(&userData); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := UpdateUserController(&user, userData, db); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User updated"})
}
func UpdateUserController(user *engine.User, userData UserData, db *gorm.DB) error {
	if !user.ADMIN {
		return errors.New("not admin")
	}
	var userDB engine.User
	fmt.Println(userData.Token)
	fmt.Println(userData.Name)
	if userData.Id != 0 {
		db.Preload("torrents").Where("id = ?", userData.Id).First(&userDB)
		if userDB.ID == 0 {
			return errors.New("user not found")
		}
	} else {
		userDB.TOKEN = userData.Token
	}
	userDB.NAME = userData.Name
	userDB.EMAIL = userData.Email
	userDB.ADMIN = userData.Admin
	userDB.CAN_DOWNLOAD = userData.CanDownload
	userDB.CAN_CONVERT = userData.CanConvert
	userDB.CAN_ADD_FILES = userData.CanAddFiles
	userDB.CAN_UPLOAD = userData.CanUpload
	if !userData.CanUpload {
		userDB.ALLOWED_UPLOAD_NUMBER = 0
		userDB.ALLOWED_UPLOAD_SIZE = 0
	}
	userDB.CAN_DELETE = userData.CanDelete
	userDB.CAN_EDIT = userData.CanEdit
	userDB.CAN_TRANSCODE = userData.CanTranscode
	if !userData.CanTranscode {
		userDB.MAX_TRANSCODING = 0
	}
	userDB.MAX_TRANSCODING = userData.MaxTranscoding
	userDB.ALLOWED_UPLOAD_NUMBER = int64(userData.AllowedUploadNumber)
	userDB.ALLOWED_UPLOAD_SIZE = int64(userData.AllowedUploadSize)
	if err := db.Save(&userDB); err.Error != nil {
		return err.Error
	}
	return nil
}

func DeleteUser(c *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, c, []string{})
	if err != nil {
		c.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	userId := c.PostForm("id")
	id, err := strconv.Atoi(userId)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if id == int(user.ID) {
		c.JSON(400, gin.H{"error": "cannot delete yourself"})
		return
	}
	if err := DeleteUserController(&user, id, db); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User deleted"})
}
func DeleteUserController(user *engine.User, userId int, db *gorm.DB) error {
	return db.Where("id = ?", userId).Delete(&engine.User{}).Error
}
