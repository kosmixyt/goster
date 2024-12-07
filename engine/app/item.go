package engine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"kosmix.fr/streaming/kosmixutil"
)

type Torrent_File struct {
	UUID  string `gorm:"primaryKey"`
	NAME  string `gorm:"not null"`
	LINK  string `gorm:"not null;unique"`
	SEED  int    `gorm:"not null"`
	LEECH int    `gorm:"not null"`
	PATH  string `gorm:"not null"`
	// provider = "sharewood" | "ygg"
	PROVIDER  string `gorm:"not null"`
	SEASON    *SEASON
	SEASON_ID *uint `gorm:"default:null"`
	TV        *TV
	TV_ID     *uint
	MOVIE     *MOVIE
	MOVIE_ID  *uint
	SIZE_str  string    `gorm:"not null"`
	LastFetch time.Time `gorm:"not null"`
	FetchData string    `gorm:"not null"`
}

func (t *Torrent_File) Load() ([]byte, error) {
	if t.PATH == "" {
		var buffer io.Reader
		var err error
		switch t.PROVIDER {
		case "sharewood":
			buffer, err = SHAREWOOD.FetchTorrentFile(t)
		case "ygg":
			buffer, err = YGG.FetchTorrentFile(t)
		case "torrent9":
			buffer, err = torrentnine.FetchTorrentFile(t)
		default:
			panic("Torrent_File.Load() called with invalid provider : " + t.PROVIDER)
		}
		if err != nil {
			return nil, err
		}
		if t.UUID == "" {
			panic("Torrent_File.Load() called with nil ID")
		}
		fileName := t.GetFileName()
		fullBuffer, err := io.ReadAll(buffer)
		if err != nil {
			return nil, err
		}
		WriteFile(bytes.NewReader(fullBuffer), Joins(FILES_TORRENT_PATH, fileName))
		t.PATH = fileName
		return fullBuffer, nil
	} else {
		fio, err := os.Open(Joins(FILES_TORRENT_PATH, t.PATH))
		if err != nil {
			fmt.Println("Error opening file", t.PATH)
			return nil, err
		}
		buff, err := io.ReadAll(fio)
		if err != nil {
			return nil, err
		}
		return buff, nil
	}
}
func (t *Torrent_File) SetAsManual(file []byte) {
	if t.UUID == "" {
		panic("Torrent_File.SetAsManual() called with nil ID")
	}
	fileName := t.GetFileName()
	WriteFile(bytes.NewReader(file), Joins(FILES_TORRENT_PATH, fileName))
	t.PATH = fileName
	t.Save()
}

func (t *Torrent_File) Save() error {
	if tx := db.Save(&t); tx.Error != nil {
		return tx.Error
	}
	return nil
}
func (t *Torrent_File) LoadMedias() {
	if t.SEASON_ID != nil {
		db.Model(&t).Association("SEASON").Find(&t.SEASON)
	}
	if t.TV_ID != nil {
		db.Model(&t).Association("TV").Find(&t.TV)
	}
	if t.MOVIE_ID != nil {
		db.Model(&t).Association("MOVIE").Find(&t.MOVIE)
	}

	if t.SEASON == nil && t.TV == nil && t.MOVIE == nil {
		fmt.Println(t.SEASON_ID, t.TV_ID, t.MOVIE_ID)
		panic("Torrent_File.LoadMedias() called with nil media")
	}
}
func (t *Torrent_File) GetMetadata() (*metainfo.Info, error) {
	buff, err := t.Load()
	if err != nil {
		return nil, err
	}
	torrent, err := metainfo.Load(bytes.NewReader(buff))
	if err != nil {
		return nil, err
	}
	info, err := torrent.UnmarshalInfo()
	if err != nil {
		return nil, err
	}
	return &info, nil
}

//	func (t *Torrent_File) GetMaxAllowedSize() int64 {
//		t.LoadMedias()
//		if t.MOVIE != nil {
//			return Config.Limits.MovieSize
//		}
//		if t.TV != nil {
//			return Config.Limits.SeasonSize
//		}
//		panic("Torrent_File.GetMaxAllowedSize() called with invalid media")
//	}
func GetMaxAllowedSize(movie *MOVIE, episode *EPISODE, season *SEASON) int64 {
	if movie != nil {
		return Config.Limits.MovieSize
	}
	if episode != nil || season != nil {
		return Config.Limits.SeasonSize
	}
	panic("GetMaxAllowedSize() called with invalid media")
}

func (t *Torrent_File) ValidateTorrent(WaitGroup *sync.WaitGroup, channel chan map[string]*Torrent_File, season *SEASON, episode *EPISODE, movie *MOVIE, i int) {
	defer WaitGroup.Done()
	info, err := t.GetMetadata()
	if err != nil {
		channel <- map[string]*Torrent_File{"Failed load metadata": t}
		return
	}
	number_media_files := 0
	if len(info.Files) == 0 {
		number_media_files = 1
	} else {
		for _, file := range info.Files {
			if kosmixutil.IsVideoFile(file.DisplayPath(info)) {
				number_media_files++
			}
		}
	}
	if info.TotalLength() == 0 {
		channel <- map[string]*Torrent_File{"Empty torrent": t}
		return
	}
	// fmt.Println("Taille du torrent", info.TotalLength(), "max", GetMaxAllowedSize(movie, episode), t.UUID)
	if info.TotalLength() > GetMaxAllowedSize(movie, episode, season) {
		channel <- map[string]*Torrent_File{"item is too large": t}
		return
	}
	if movie != nil {
		if number_media_files != 1 {
			fmt.Println("Number of files", number_media_files, len(info.Files))
			channel <- map[string]*Torrent_File{"Movie must have only one file": t}
			return
		}
		channel <- map[string]*Torrent_File{"": t}
		return
	} else {
		if number_media_files != len(season.EPISODES) {
			// if number of files is different for the number of episodes
			// channel <- map[bool]provider.TorrentItem{false: item}
		}
		haveWant := false
		for _, file := range info.Files {
			if !kosmixutil.IsVideoFile(file.DisplayPath(info)) {
				continue
			}
			name := filepath.Base(file.DisplayPath(info))
			episode_number, _ := kosmixutil.GetEpisode(name)
			if episode_number == 0 {
				channel <- map[string]*Torrent_File{"Failed determine episode number for " + file.DisplayPath(info): t}
				return
			}
			if episode != nil {
				if episode_number == episode.NUMBER {
					haveWant = true
				}
			}
		}
		if haveWant || episode == nil {
			channel <- map[string]*Torrent_File{"": t}
			return
		}
		channel <- map[string]*Torrent_File{"Episode not found in serie": t}
		return
	}

}

func (t *Torrent_File) GetFileName() string {
	fileName := t.PROVIDER + "-" + t.UUID + ".torrent"
	return fileName
}
