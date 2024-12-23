package engine

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"kosmix.fr/streaming/engine/storage"
	"kosmix.fr/streaming/kosmixutil"
)

type StorageDbElement struct {
	gorm.Model
	Name  string                `gorm:"not null"`
	ID    uint                  `gorm:"unique;not null,primary_key"`
	Paths []*StoragePathElement `gorm:"foreignKey:StorageId;constraint:OnDelete:CASCADE"`
}
type StoragePathElement struct {
	gorm.Model
	StorageId uint
	// path
	Path    string
	Size    int64
	Storage *StorageDbElement `gorm:"foreignKey:StorageId"`
	Files   []*FILE           `gorm:"foreignKey:STORAGE_ID;constraint:OnDelete:CASCADE"`
	Records []*Record         `gorm:"constraint:OnDelete:CASCADE;foreignKey:OutputPathStorerId;"`
}

func (s *StoragePathElement) getStorage() *StorageDbElement {
	if s.Storage == nil {
		db.Model(s).Association("Storage").Find(&s.Storage)
	}
	if s.Storage == nil {
		fmt.Println("Storage not must be torrent_path")
	}
	return s.Storage
}

func (s *StoragePathElement) toStorage() kosmixutil.PathElement {
	return kosmixutil.PathElement{
		Path: s.Path,
		Size: s.Size,
	}
}
func (s *StorageDbElement) toConn() storage.Storage {
	conn := GetStorageConFromId(s.ID)
	if conn == nil {
		panic("Storage not found")
	}
	return conn.Conn
}
func (s *StorageDbElement) HasRootPath(path string) bool {
	s.LoadPaths()
	for _, p := range s.Paths {
		if p.Path == path {
			return true
		}
	}
	return false
}
func (s *StorageDbElement) GetRootPath(path string) (*StoragePathElement, error) {
	s.LoadPaths()
	for _, p := range s.Paths {
		if p.Path == path {
			return p, nil
		}
	}
	return nil, errors.New("path not found")
}
func (s *StorageDbElement) LoadPaths() ([]*StoragePathElement, error) {
	var paths []*StoragePathElement
	db.Model(s).Association("Paths").Find(&paths)
	return paths, nil
}

func DispatchStorage(TYPE string) (storage.Storage, error) {
	switch TYPE {
	case "sftp":
		return &storage.SftpStorage{}, nil
	case "local":
		return &storage.LocalStorage{}, nil
	default:
		return nil, errors.New("Invalid storage type")
	}
}
func ParsePath(bundledPath string) (*MemoryStorage, string, error) {
	elements := strings.Split(bundledPath, "@")
	if len(elements) != 2 {
		return nil, "", errors.New("Invalid path")
	}
	id, err := strconv.Atoi(elements[0])
	if err != nil {
		return nil, "", err
	}
	storage := GetStorageConFromId(uint(id))
	if storage == nil {
		return nil, "", errors.New("Storage not found")
	}
	pathsOfStorage := storage.DbElement.PathAsString()
	if !slices.Contains(pathsOfStorage, elements[1]) {
		return nil, "", errors.New("Invalid path")
	}
	return storage, elements[1], nil
}

type StoragesRender struct {
	ID    uint     `json:"id"`
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

func GetStorageRenders() ([]StoragesRender, error) {
	var storages []StorageDbElement
	if tx := db.Preload("Paths").Find(&storages); tx.Error != nil {
		return nil, tx.Error
	}
	var paths []StoragesRender
	for _, storage := range storages {
		paths = append(paths, StoragesRender{
			ID:    storage.ID,
			Name:  storage.Name,
			Paths: storage.PathAsString(),
		})
	}
	return paths, nil
}

func (s *StorageDbElement) PathAsString() []string {
	s.LoadPaths()
	var paths []string = make([]string, 0)
	for _, path := range s.Paths {
		paths = append(paths, path.Path)
	}
	return paths
}
