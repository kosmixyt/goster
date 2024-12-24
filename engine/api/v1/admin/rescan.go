package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

func RescanController(user *engine.User, db *gorm.DB) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	return engine.Scan(db)
}

func Rescan(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	err = RescanController(&user, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "ok"})
}

func GetStorages(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(401, gin.H{"error": "not admin"})
		return
	}
	storage := GetStoragesController(db)
	ctx.JSON(200, storage)
}
func GetStoragesController(db *gorm.DB) []Storage {
	var storages []engine.StorageDbElement
	var renders []Storage
	db.Find(&storages)
	for _, s := range storages {
		mem := engine.GetStorageConFromId(s.ID)
		item := Storage{
			ID:      s.ID,
			Name:    s.Name,
			TYPE:    mem.Conn.Type(),
			OPTIONS: map[string]string{},
		}
		for _, PathElement := range mem.DbElement.Paths {
			item.Paths = append(item.Paths, Path{
				ID:   PathElement.ID,
				Path: PathElement.Path,
				Size: PathElement.Size,
			})
		}
		renders = append(renders, item)
	}
	return renders
}

type Storage struct {
	ID      uint              `json:"id"`
	Name    string            `json:"name"`
	TYPE    string            `json:"type"`
	OPTIONS map[string]string `json:"options"`
	Paths   []Path            `json:"paths"`
}
type Path struct {
	ID   uint   `json:"id"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func ListDir(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	target_storage_id, err := strconv.Atoi(ctx.Query("target_storage_id"))
	path := ctx.Query("path")
	if err != nil {
		ctx.JSON(400, gin.H{"error": "target_storage_id is not a number"})
		return
	}
	storage := engine.GetStorageConFromId(uint(target_storage_id))
	if storage == nil {
		ctx.JSON(400, gin.H{"error": "storage not found"})
		return
	}
	paths, err := ListDirController(&user, storage, path)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, paths)
}
func ListDirController(user *engine.User, target_storage *engine.MemoryStorage, path string) ([]string, error) {
	if !user.ADMIN {
		return nil, engine.ErrorIsNotAdmin
	}
	paths, err := target_storage.Conn.ListDir(path)
	if err != nil {
		return nil, err

	}
	strpaths := []string{}
	for _, p := range paths {
		if p.IsDir() {
			strpaths = append(strpaths, p.Name())
		}
	}
	return strpaths, nil
}

func DeleteStorage(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	name := ctx.Query("name")
	err = DeleteStorageController(&user, name)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "Storage has been deleted, please reboot to apply changes."})
}
func DeleteStorageController(user *engine.User, name string) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	if err := engine.DeleteStorage(name); err != nil {
		return err
	}
	return nil
}

type CreateStoragesRequest struct {
	Name    string            `json:"name"`
	TYPE    string            `json:"type"`
	OPTIONS map[string]string `json:"options"`
}

func CreateStorage(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var payload CreateStoragesRequest
	err = ctx.BindJSON(&payload)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = CreateStorageController(&user, payload)
}

func CreateStorageController(user *engine.User, payload CreateStoragesRequest) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	return nil
}

func AddPath(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var payloadPath kosmixutil.PathElement
	err = ctx.BindJSON(&payloadPath)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = AddPathController(&user, ctx.Query("storage_name"), payloadPath)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "Path has been added to storage, please reboot to apply changes."})
}
func AddPathController(user *engine.User, storage_name string, path kosmixutil.PathElement) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	return engine.AddPath(storage_name, path)
}
func DeletePath(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	var payloadPath kosmixutil.PathElement
	err = ctx.BindJSON(&payloadPath)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = DeletePathController(&user, ctx.Query("storage_name"), payloadPath)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "Path has been deleted from storage, please reboot to apply changes."})
}

func DeletePathController(user *engine.User, storage_name string, path kosmixutil.PathElement) error {
	if !user.ADMIN {
		return engine.ErrorIsNotAdmin
	}
	return engine.DeletePath(storage_name, path)
}
