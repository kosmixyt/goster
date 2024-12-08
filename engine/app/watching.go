package engine

import (
	"fmt"

	"gorm.io/gorm"
)

type WATCHING struct {
	gorm.Model
	ID         uint     `gorm:"unique;not null,primary_key"` // use
	CURRENT    int64    `gorm:"not null"`
	TOTAL      int64    `gorm:"not null"`
	FILE       FILE     `gorm:"foreignKey:FILE_ID"`
	FILE_ID    uint     `gorm:"not null"`
	USER       *User    `gorm:"foreignKey:USER_ID"`
	USER_ID    uint     `gorm:"not null"`
	MOVIE      *MOVIE   `gorm:"foreignKey:MOVIE_ID"`
	MOVIE_ID   uint     `gorm:"default:null"`
	TV         *TV      `gorm:"foreignKey:TV_ID"`
	TV_ID      uint     `gorm:"default:null"`
	EPISODE    *EPISODE `gorm:"foreignKey:EPISODE_ID"`
	EPISODE_ID *uint    `gorm:"default:null"`
}

func (w *WATCHING) GetNextFile() *SKINNY_RENDER {
	if w.USER == nil {
		panic("Invalid user")
	}
	if w.MOVIE != nil {
		similars := w.MOVIE.SimilarMovies(w.USER.SkinnyMoviePreloads, 4)
		if len(similars) == 0 {
			fmt.Println("[WARN] No similar movie found for movie", w.MOVIE.ID)
			return &SKINNY_RENDER{}
		}
		sim := similars[0].Skinny(w)
		return &sim
	}
	if w.TV != nil {
		WatchingOfEpisode := w.EPISODE
		seasons := w.TV.SEASON
		for _, s := range seasons {
			for _, e := range s.EPISODES {
				if e.ID == WatchingOfEpisode.ID {
					continue
				}
				if e.NUMBER > WatchingOfEpisode.NUMBER {
					// return &SKINNY_RENDER{}
					// next
				}
			}
		}
		return &SKINNY_RENDER{}
	}
	fmt.Println("[WARN] No media type found for watching", w.ID)
	panic("Invalid type")
}
func (w *WATCHING) ToSkinny() SKINNY_RENDER {
	if w.MOVIE != nil {
		return w.MOVIE.Skinny(w)
	}
	if w.TV != nil {
		data := w.TV.Skinny(w)
		data.DisplayData = "S" + w.EPISODE.SEASON.GetNumberAsString(true) + "E" + w.EPISODE.GetNumberAsString(true)
		if w.EPISODE.NAME != "" {
			data.DisplayData += " - " + w.EPISODE.NAME
		}
		return data
	}
	panic("Invalid type")
}
func (w *WATCHING) WatchData() WatchData {
	if w == nil {
		return WatchData{TOTAL: 0, CURRENT: 0}
	}
	return WatchData{TOTAL: w.TOTAL, CURRENT: w.CURRENT}
}

func MapWatching(w1 []WATCHING) []SKINNY_RENDER {
	var res []SKINNY_RENDER = make([]SKINNY_RENDER, 0)
	for _, w := range w1 {
		res = append(res, w.ToSkinny())
	}
	return res
}
