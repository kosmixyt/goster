package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"kosmix.fr/streaming/kosmixutil"
)

type SftpStorage struct {
	sftpClient *sftp.Client
	props      SftpStorageProps
	paths      []kosmixutil.PathElement
	name       string
}

type SftpStorageProps struct {
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
	// Path []PathElement `json:"path"`
}

func (s *SftpStorage) Init(name string, channel chan error, props interface{}, paths []kosmixutil.PathElement) {
	s.paths = paths
	jsonData, err := json.Marshal(props)
	if err != nil {
		channel <- err
	}
	var data SftpStorageProps
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		channel <- err
	}
	for _, path := range paths {
		if strings.Contains(path.Path, "@") {
			channel <- errors.New("path cannot contain @")
		}
	}
	config := &ssh.ClientConfig{
		User: data.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(data.Pass),
		},
		Timeout:         time.Second * 40,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshAdress := fmt.Sprintf("%s:%s", data.Host, data.Port)
	client, err := ssh.Dial("tcp", sshAdress, config)
	if err != nil {
		channel <- err
	}
	// defer client.Close()
	// fmt.Println("Connected to sftp", data.User, data.Pass)
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		channel <- errors.New("cannot create sftp client verify your credentials")
	}
	s.sftpClient = sftpClient

	s.props = data
	s.name = name
	channel <- nil
}

func (s *SftpStorage) NeedProxy() bool {
	return true
}

func (s *SftpStorage) GetReader(path string) (io.ReadSeekCloser, error) {
	return s.sftpClient.Open(path)
}
func (s *SftpStorage) GetFreeSpace(path string) (uint64, error) {
	return 0, errors.New("cannot get free space on sftp")
}
func (s *SftpStorage) Paths() []kosmixutil.PathElement {
	return s.paths
}
func (s *SftpStorage) TransferSpeed() int {
	return 0
}

func (s *SftpStorage) GetFfmpegUrl(path string) (string, bool) {
	return "", s.NeedProxy()
}
func (s *SftpStorage) Stats(path string) (os.FileInfo, error) {
	return s.sftpClient.Stat(path)
}

type FileData struct {
	Path       string
	FileName   string
	StorerDbId uint
	Size       int64
	ROOT_PATH  interface{}
}

func (s *SftpStorage) RecursiveScan(path kosmixutil.PathElement) ([]FileData, error) {
	var files []FileData
	walker := s.sftpClient.Walk(path.Path)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return nil, err
		}
		if !walker.Stat().IsDir() {
			files = append(files, FileData{
				Path:     walker.Path(),
				FileName: walker.Stat().Name(),
				Size:     walker.Stat().Size(),
			})
		}
	}
	return files, nil
}
func (s *SftpStorage) Name() string {
	return s.name
}

func (s *SftpStorage) Remove(path string) error {
	return s.sftpClient.Remove(path)
}
func (s *SftpStorage) GetWriter(path string) (io.WriteCloser, error) {
	return s.sftpClient.Create(path)
}

func (s *SftpStorage) Rename(oldPath string, newPath string) error {
	return s.sftpClient.Rename(oldPath, newPath)
}

func (s *SftpStorage) Type() string {
	return "sftp"
}

func (s SftpStorage) ListDir(path string) ([]os.FileInfo, error) {
	return s.sftpClient.ReadDir(path)
}
