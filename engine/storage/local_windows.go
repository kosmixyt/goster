//go:build windows
// +build windows

package storage

import (
	"fmt"
)

func (l *LocalStorage) GetFreeSpace(path string) (uint64, error) {
	fmt.Println("path", path)
	return 0, fmt.Errorf("cannot get free space on local storage on windows")
}
func GetAvailableSizeInDownloadPath(path string) (uint64, error) {
	return 0, fmt.Errorf("cannot get free space on local storage on windows")
}
