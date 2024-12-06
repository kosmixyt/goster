//go:build linux
// +build linux

package storage

import "syscall"

func (l *LocalStorage) GetFreeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		panic(err)
	}
	freeSpace := stat.Bavail * uint64(stat.Bsize)
	return freeSpace, nil
}
func GetAvailableSizeInDownloadPath(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Stat
	if err != nil {
		panic(err)
	}
	freeSpace := stat.Bavail * uint64(stat.Bsize)
	return freeSpace, nil
}
