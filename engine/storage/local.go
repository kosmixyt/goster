package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kosmix.fr/streaming/kosmixutil"
)

type LocalStorage struct {
	props LocalStorageProps
	name  string
	paths []kosmixutil.PathElement
}
type LocalStorageProps struct {
	// Path []PathElement `json:"path"`
	// TorrentPath string   `json:"torrent_path"`
}

func (l *LocalStorage) Init(name string, channel chan error, props interface{}, paths []kosmixutil.PathElement) {
	l.paths = paths
	jsonData, err := json.Marshal(props)
	if err != nil {
		channel <- err
	}
	var data LocalStorageProps
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		channel <- err
	}
	for _, path := range paths {
		if strings.Contains(path.Path, "@") {
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

func (l *LocalStorage) Paths() []kosmixutil.PathElement {
	// fmt.Println("l.props.Path", l.props.Path)
	return l.paths
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

func (l *LocalStorage) RecursiveScan(path kosmixutil.PathElement) ([]FileData, error) {
	var files []FileData
	err := filepath.Walk(path.Path, func(path string, info os.FileInfo, err error) error {
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

func (l *LocalStorage) Type() string {
	return "local"
}
func (l *LocalStorage) ListDir(path string) ([]os.FileInfo, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var infos []os.FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}
func (l *LocalStorage) Close() error {
	return nil
}
