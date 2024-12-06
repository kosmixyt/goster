package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	props LocalStorageProps
	name  string
}
type LocalStorageProps struct {
	Path []string `json:"path"`
	// TorrentPath string   `json:"torrent_path"`
}

func (l *LocalStorage) Init(name string, channel chan error, props interface{}) {
	jsonData, err := json.Marshal(props)
	if err != nil {
		channel <- err
	}
	var data LocalStorageProps
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		channel <- err
	}
	for _, path := range data.Path {
		if strings.Contains(path, "@") {
			channel <- errors.New("path cannot contain @")
		}
	}
	l.props = data
	l.name = "Local Storage"
	channel <- nil
}

func (l *LocalStorage) GetReader(path string) (io.ReadSeekCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (l *LocalStorage) Paths() []string {
	return l.props.Path
}

func (l *LocalStorage) GetFfmpegUrl(path string) (string, bool) {
	fmt.Println("path", path)
	return path, l.NeedProxy()
}
func (l *LocalStorage) Stats(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (l *LocalStorage) NeedProxy() bool {
	return false
}

func (l *LocalStorage) RecursiveScan(path string) ([]FileData, error) {
	var files []FileData
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, FileData{
			Path:     path,
			FileName: info.Name(),
			Size:     info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (l *LocalStorage) Remove(path string) error {
	return os.Remove(path)
}

func (l *LocalStorage) Name() string {
	return l.name
}
func (l *LocalStorage) GetWriter(path string) (io.WriteCloser, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (l *LocalStorage) Rename(oldPath string, newPath string) error {
	return os.Rename(oldPath, newPath)
}
