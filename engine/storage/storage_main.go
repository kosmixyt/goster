package storage

import (
	"io"
	"os"

	"kosmix.fr/streaming/kosmixutil"
)

type Storage interface {
	Init(string, chan error, interface{}, []kosmixutil.PathElement)
	GetReader(path string) (io.ReadSeekCloser, error)
	GetFreeSpace(path string) (uint64, error)
	Paths() []kosmixutil.PathElement
	GetFfmpegUrl(path string) (string, bool)
	RecursiveScan(path kosmixutil.PathElement) ([]FileData, error)
	Stats(path string) (os.FileInfo, error)
	Remove(path string) error
	Name() string
	GetWriter(path string) (io.WriteCloser, error)
	NeedProxy() bool
	Rename(oldPath string, newPath string) error
	Type() string
	ListDir(path string) ([]os.FileInfo, error)
	Close() error
}
