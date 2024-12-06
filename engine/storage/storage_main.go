package storage

import (
	"io"
	"os"
)

type Storage interface {
	Init(string, chan error, interface{})
	GetReader(path string) (io.ReadSeekCloser, error)
	GetFreeSpace(path string) (uint64, error)
	Paths() []string
	GetFfmpegUrl(path string) (string, bool)
	RecursiveScan(path string) ([]FileData, error)
	Stats(path string) (os.FileInfo, error)
	Remove(path string) error
	Name() string
	GetWriter(path string) (io.WriteCloser, error)
	NeedProxy() bool
	Rename(oldPath string, newPath string) error
}
