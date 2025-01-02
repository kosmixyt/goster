package engine

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DownloadRequest struct {
	gorm.Model
	MAX_SIZE     int64
	STATUS       string    // pending, finished, error
	OWNER        *User     `gorm:"foreignKey:OWNER_ID"`
	OWNER_ID     uint      `gorm:"not null"`
	Interval     int       `gorm:"default:24"`
	Movie        *MOVIE    `gorm:"foreignKey:MOVIE_ID"`
	MOVIE_ID     uint      `gorm:"default:null"`
	TV_SEASON    *SEASON   `gorm:"foreignKey:TV_SEASON_ID"`
	TV_SEASON_ID uint      `gorm:"default:null"`
	TV           *TV       `gorm:"foreignKey:TV_ID"`
	TV_ID        uint      `gorm:"default:null"`
	LAST_TRY     time.Time `gorm:"default:null"`
	TORRENT      *Torrent  `gorm:"foreignKey:TORRENT_ID"`
	TORRENT_ID   uint      `gorm:"default:null"`
	ERROR        string
}

func (user *User) NewRequestDownload(max_size int64, Movie *MOVIE, TV_SEASON *SEASON, TV *TV) *DownloadRequest {
	var request = &DownloadRequest{
		MAX_SIZE:   max_size,
		Interval:   Config.Limits.CheckInterval,
		STATUS:     "pending",
		ERROR:      "",
		OWNER:      user,
		OWNER_ID:   user.ID,
		Movie:      Movie,
		TV_SEASON:  TV_SEASON,
		TV:         TV,
		TORRENT:    nil,
		TORRENT_ID: 0,
	}
	if Movie != nil {
		request.MOVIE_ID = Movie.ID
	}
	if TV_SEASON != nil {
		request.TV_SEASON_ID = TV_SEASON.ID
		request.TV_ID = TV.ID
	}

	db.Create(request)
	return request
}
func InitIntervals(db *gorm.DB) {
	for {
		var requests []DownloadRequest
		db.Preload("TV").Preload("TV_SEASON").Preload("Movie").Preload("OWNER").Where("STATUS = ?", "pending").Find(&requests)
		for _, req := range requests {
			if req.STATUS != "pending" {
				continue
			}
			if err := req.Check(); err != nil {
				fmt.Println("Error in request", err.Error())
			}
			req.LAST_TRY = time.Now()
			db.Where("id = ?", req.ID).Model(&DownloadRequest{}).Updates(map[string]interface{}{"last_try": time.Now()})
		}
		time.Sleep(time.Duration(Config.Limits.CheckInterval) * time.Hour)
	}
}

func (req *DownloadRequest) Check() error {
	if req.STATUS != "pending" {
		panic("Request is not pending")
	}
	if req.Movie != nil {
		req.Movie.LoadFiles(db)
		if req.Movie.HasFile(nil) {
			req.STATUS = "finished"
			db.Model(req).Updates(map[string]interface{}{"STATUS": req.STATUS})
			return errors.New("already downloaded")
		}
	} else if req.TV_SEASON != nil {
		req.TV_SEASON.Refresh(func() *gorm.DB {
			return db.Preload("EPISODES").Preload("EPISODES.FILES")
		})
		if req.TV_SEASON.HasFile() {
			req.STATUS = "finished"
			db.Model(req).Updates(map[string]interface{}{"STATUS": req.STATUS})
			return errors.New("already downloaded")
		}
	}
	ordered_providers, err := FindBestTorrentFor(req.TV, req.Movie, req.TV_SEASON, nil, 1, req.MAX_SIZE)
	if err != nil {
		return err
	}
	for _, provider := range ordered_providers {
		item, err := AddTorrent(req.OWNER, provider, GetMediaId(req.Movie, req.TV), GetType(req.TV, req.Movie), req.MAX_SIZE, func(s string) {})
		if err != nil {
			if err.UserLimit {
				req.SetAsError(err.Err)
				return err.Err
			} else {
				continue
			}
		}
		RegisterFilesInTorrent(item, req.Movie, req.TV_SEASON)
		req.STATUS = "finished"
		db.Model(req).Updates(map[string]interface{}{"STATUS": req.STATUS, "TORRENT_ID": item.DB_ITEM.ID})
		return nil
	}
	return errors.New("no torrent found")
}
func (req *DownloadRequest) SetAsError(err error) {
	req.ERROR = err.Error()
	req.STATUS = "error"
	db.Save(req)
}
