package engine

import (
	"fmt"
	"io"
	"strconv"

	"gorm.io/gorm"
)

// this function only add log to the task but dont save it / set error
func GetMediaReader(db *gorm.DB, user *User, mediaType string, provider string, id int, season int, episode int, preferedTorrent_file_id string, task *Task, progress func(string)) (*FILE, error) {
	if mediaType == Tv {
		var tv TV
		if tx := db.Where(provider+" = ?", id).Preload("SEASON").Preload("SEASON.EPISODES").Preload("SEASON.EPISODES.FILES").Preload("FILES").First(&tv); tx.Error != nil {
			return nil, tx.Error
		}
		if season == 0 || episode == 0 {
			return nil, fmt.Errorf("invalid season or episode")
		}
		SeasonElement := tv.GetSeason(season, false)
		if SeasonElement == nil {
			return nil, fmt.Errorf("season not found in serie")
		}
		EpisodeElement := SeasonElement.GetEpisode(episode, false)
		if EpisodeElement == nil {
			return nil, fmt.Errorf("episode not found in serie")
		}
		if EpisodeElement.HasFile(nil) {
			progress("Episode already downloaded file")
			task.AddLog("Episode already downloaded file=", strconv.Itoa(int(EpisodeElement.FILES[0].ID)))
			return &EpisodeElement.FILES[0], nil
		}
		if SeasonElement.HasFile() {
			return nil, fmt.Errorf("season already downloaded")
		}
		renderChannel := make(chan *FILE, 1)
		errorChannel := make(chan error, 1)
		task.UpdateName(" AND DOWNLOADING " + tv.NAME + " S" + fmt.Sprint(season) + "E" + fmt.Sprint(episode))
		task.AddLog("Searching for torrent")
		go SearchAndDownloadTorrent(db, user, nil, EpisodeElement, SeasonElement, &tv, preferedTorrent_file_id, renderChannel, errorChannel, task, progress)
		return <-renderChannel, <-errorChannel
	}
	if mediaType == Movie {
		var movie MOVIE
		if tx := db.Where(provider+" = ?", id).Preload("FILES").First(&movie); tx.Error != nil {
			return nil, tx.Error
		}
		if movie.HasFile(nil) {
			progress("Movie already downloaded : " + movie.FILES[0].FILENAME)
			task.AddLog("Movie already downloaded file=", strconv.Itoa(int(movie.FILES[0].ID)))
			return &movie.FILES[0], nil
		}
		renderChannel := make(chan *FILE, 1)
		errorChannel := make(chan error, 1)
		task.UpdateName(" AND DOWNLOADING " + movie.NAME)
		go SearchAndDownloadTorrent(db, user, &movie, nil, nil, nil, (preferedTorrent_file_id), renderChannel, errorChannel, task, progress)
		return <-renderChannel, <-errorChannel
	}
	panic("not implemented")
}

var Uploads []*Upload = make([]*Upload, 0)

// an upload is obligatory in a root_path
type Upload struct {
	ID      uint           `gorm:"primaryKey"`
	USER_ID uint           `gorm:"index"`
	USER    *User          `gorm:"foreignKey:USER_ID"`
	Writer  io.WriteCloser `gorm:"-"`
	Name    string
	Storer  *MemoryStorage `gorm:"-"`
	// !! root_path of storer
	Storer_path string
	CURRENT     int64
	TOTAL       int64
	EPISODE     *EPISODE `gorm:"-"`
	MOVIE       *MOVIE   `gorm:"-"`
}

func (u *Upload) Write(p []byte) (err error) {
	n := len(p)
	u.CURRENT += int64(n)
	if u.CURRENT > u.TOTAL {
		return fmt.Errorf("out of bounds")
	}
	if n, err = u.Writer.Write(p); err != nil {
		fmt.Println("error writing to file", err)
		return err
	} else {
		fmt.Println("writing to file", n)
	}
	if u.CURRENT == u.TOTAL {
		u.Writer.Close()
		fmt.Println("upload done")
		return u.End()
	}
	return nil
}
func (u *Upload) End() error {
	file := &FILE{
		FILENAME:  u.Name,
		ROOT_PATH: u.Storer_path,
		SUB_PATH:  "",
		STORAGEID: &u.Storer.DbElement.ID,
		SIZE:      u.TOTAL,
		IS_MEDIA:  true,
		SHARES:    make([]Share, 0),
		WATCHING:  make([]WATCHING, 0),
	}
	if u.MOVIE != nil {
		file.MOVIE = u.MOVIE
		file.MOVIE_ID = u.MOVIE.ID
	}
	if u.EPISODE != nil {
		u.EPISODE.LoadSeason()
		file.EPISODE_ID = u.EPISODE.ID
		file.SEASON_ID = u.EPISODE.SEASON.ID
		file.TV_ID = u.EPISODE.SEASON.TV_ID
	}
	if err := db.Create(file).Error; err != nil {
		return err
	}
	// delete u from array uploads
	for i, up := range Uploads {
		if up == u {
			Uploads = append(Uploads[:i], Uploads[i+1:]...)
			break
		}
	}
	return nil
}
