package engine

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

type SEASON struct {
	gorm.Model
	ID                  uint       `gorm:"unique;not null,primary_key"` // use
	EPISODES            []*EPISODE `gorm:"foreignKey:SEASON_ID"`
	TV_ID               uint
	TV                  TV             `gorm:"foreignKey:TV_ID"`
	NAME                string         `gorm:"not null"`
	NUMBER              int            `gorm:"not null"`
	DESCRIPTION         string         `gorm:"not null"`
	BACKDROP_IMAGE_PATH string         `gorm:"not null"`
	TORRENT_FILES       []Torrent_File `gorm:"foreignKey:SEASON_ID"`
}

func (s *SEASON) GetNumberAsString(withC bool) string {
	c := ""
	if s.NUMBER < 10 && withC {
		c = "0"
	}
	return c + strconv.Itoa(s.NUMBER)
}
func (s *SEASON) HasFile() bool {
	if len(s.EPISODES) == 0 {
		fmt.Println("[WARN] No episode found for season", s.ID)
	}
	for _, episode := range s.EPISODES {
		if len(episode.FILES) > 0 {
			return true
		}
	}
	return false
}
func (s *SEASON) Refresh(preload func() *gorm.DB) {
	if err := preload().Where("id = ?", s.ID).First(s).Error; err != nil {
		panic(err)
	}
}

func (s *SEASON) GetEpisode(episode int, createIfNotExist bool) *EPISODE {
	if len(s.EPISODES) == 0 {
		fmt.Println("[WARN] No episode found for season", s.ID)
	}
	for _, ep := range s.EPISODES {
		if ep.NUMBER == episode {
			return ep
		}
	}
	if !createIfNotExist {
		return nil
	}
	episodeElement := &EPISODE{
		NUMBER:                   episode,
		SEASON_ID:                (s.ID),
		SEASON:                   s,
		NAME:                     fmt.Sprintf("Episode %d", episode),
		FILES:                    []FILE{},
		DESCRIPTION:              fmt.Sprintf("Description of episode %d", episode),
		STILL_IMAGE_PATH:         "",
		STILL_IMAGE_STORAGE_TYPE: 2,
	}
	db.Save(&episodeElement)
	s.EPISODES = append(s.EPISODES, episodeElement)
	return episodeElement
}

func (s *SEASON) ToEpisode() []EpisodeItem {
	from := SortEpisodeByNumber(s.EPISODES)
	episodes := []EpisodeItem{}
	for _, episode := range from {
		newEpisodeItem := EpisodeItem{
			ID:             episode.ID,
			EPISODE_NUMBER: episode.NUMBER,
			FILES:          episode.ToFile(s),
			NAME:           episode.NAME,
			DESCRIPTION:    episode.DESCRIPTION,
			TRANSCODE_URL:  fmt.Sprintf(Config.Web.PublicUrl+"/transcode?type=tv&id=db@%d&season=%d&episode=%d", s.TV_ID, s.NUMBER, episode.NUMBER),
			DOWNLOAD_URL:   fmt.Sprintf(Config.Web.PublicUrl+"/download?type=tv&id=db@%d&season=%d&episode=%d", s.TV_ID, s.NUMBER, episode.NUMBER),
		}
		if (episode.WATCHING) != nil {
			if len(episode.WATCHING) > 0 {
				newEpisodeItem.WATCH = WatchData{TOTAL: episode.WATCHING[0].TOTAL, CURRENT: episode.WATCHING[0].CURRENT}
			}
		}
		switch episode.STILL_IMAGE_STORAGE_TYPE {
		case 1:
			newEpisodeItem.STILL = TMDB_LOW + episode.STILL_IMAGE_PATH
		case 0:
			newEpisodeItem.STILL = fmt.Sprintf(Config.Web.PublicUrl+"/image?type=tv&id=%d&season=%d&episode=%d&image=still", s.TV_ID, s.NUMBER, episode.NUMBER)
		}
		episodes = append(episodes, newEpisodeItem)
	}

	return episodes
}
func (s *SEASON) GetExistantEpisodeById(id uint) *EPISODE {
	for _, episode := range s.EPISODES {
		if episode.ID == id {
			return episode
		}
	}
	return nil
}

func (s *SEASON) GetSearchName(tv *TV) []string {
	j := s.GetNumberAsString(true)
	return []string{
		tv.NAME + " S" + j,
		tv.ORIGINAL_NAME + " S" + j,
	}
}
