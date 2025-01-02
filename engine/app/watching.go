package engine

import (
	"fmt"
	"gorm.io/gorm"
	"sort"
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

func (w *WATCHING) GetNextFile() *NextFile {
	if w.USER == nil {
		panic("Invalid user")
	}
	if w.MOVIE != nil {
		similars := w.MOVIE.SimilarMovies(w.USER.SkinnyMoviePreloads, 4)
		if len(similars) == 0 {
			fmt.Println("[WARN] No similar movie found for movie", w.MOVIE.ID)
			return nil
		}
		sim := similars[0]
		if !sim.HasFile(nil) {
			fmt.Println("[WARN] No file found for similar movie", sim.ID)
			return nil
		}
		return sim.FILES[0].toNextFile(nil, &sim)
	}
	if w.TV != nil {
		next := w.GetNextEpisode()
		if next == nil {
			return nil
		}
		if !next.HasFile(nil) {
			fmt.Println("[WARN] No file found for next episode", next.ID)
			return nil
		}
		return next.FILES[0].toNextFile(next, nil)
	}
	fmt.Println("[WARN] No media type found for watching", w.ID)
	panic("Invalid type")
}
func (w *WATCHING) GetNextEpisode() *EPISODE {
	if w.EPISODE == nil || w.TV == nil {
		return nil
	}
	currentSeason := w.EPISODE.SEASON
	currentEpisodeNumber := w.EPISODE.NUMBER
	// Trier les épisodes de la saison actuelle
	sort.Slice(currentSeason.EPISODES, func(i, j int) bool {
		return currentSeason.EPISODES[i].NUMBER < currentSeason.EPISODES[j].NUMBER
	})
	// Trouver le prochain épisode dans la même saison
	for _, episode := range currentSeason.EPISODES {
		if episode.NUMBER > currentEpisodeNumber {
			return episode
		}
	}
	// Trier les saisons
	sort.Slice(w.TV.SEASON, func(i, j int) bool {
		return w.TV.SEASON[i].NUMBER < w.TV.SEASON[j].NUMBER
	})

	// Trouver le prochain épisode dans les saisons suivantes
	for _, season := range w.TV.SEASON {
		if season.NUMBER > currentSeason.NUMBER {
			sort.Slice(season.EPISODES, func(i, j int) bool {
				return season.EPISODES[i].NUMBER < season.EPISODES[j].NUMBER
			})
			if len(season.EPISODES) > 0 {
				return season.EPISODES[0]
			}
		}
	}
	// Aucun épisode suivant trouvé
	return nil
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
