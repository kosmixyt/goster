package engine

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"kosmix.fr/streaming/engine/storage"
)

type StorageDbElement struct {
	Name    string   `gorm:"not null"`
	ID      uint     `gorm:"unique;not null,primary_key"`
	FILES   []FILE   `gorm:"foreignKey:STORAGEID;constraint:OnDelete:CASCADE"`
	Roots   string   `gorm:"not null"`
	Records []Record `gorm:"foreignKey:OutputStorerId;constraint:OnDelete:CASCADE"`
}

func (s *StorageDbElement) toConn() storage.Storage {
	conn := GetStorageConFromId(s.ID)
	if conn == nil {
		panic("Storage not found")
	}
	return conn.Conn
}
func (s *StorageDbElement) HasRootPath(path string) bool {
	return slices.Contains(strings.Split(s.Roots, ","), path)
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
	pathsOfStorage := strings.Split(storage.DbElement.Roots, ",")
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
	if tx := db.Find(&storages); tx.Error != nil {
		return nil, tx.Error
	}
	var paths []StoragesRender
	for _, storage := range storages {
		paths = append(paths, StoragesRender{
			ID:    storage.ID,
			Name:  storage.Name,
			Paths: strings.Split(storage.Roots, ","),
		})
	}
	return paths, nil
}
