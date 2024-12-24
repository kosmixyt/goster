package admin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
)

func TorrentPage(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	items, err := TorrentPageController(&user, db)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, items)
}

func TorrentPageController(user *engine.User, db *gorm.DB) (*TorrentPageInfo, error) {
	if !user.ADMIN {
		return nil, errors.New("unauthorized")
	}
	items := &TorrentPageInfo{
		DownloadPath: engine.Config.Torrents.DownloadPath,
		MovieSize:    engine.Config.Limits.MovieSize,
		SeasonSize:   engine.Config.Limits.SeasonSize,
		Cloudflare: Cloudflare{
			ChallengeResolver:  engine.Config.Cloudflare.ChallengeResolver,
			FlaresolverrUrl:    engine.Config.Cloudflare.FlaresolverrUrl,
			CapsolverrProxyUrl: engine.Config.Cloudflare.CapsolverrProxyUrl,
			CapsolverrApiKey:   engine.Config.Cloudflare.CapsolverrApiKey,
		},
		Providers: []TorrentProvider{},
	}
	for name, provider := range engine.Providers {
		items.Providers = append(items.Providers, TorrentProvider{
			Name:             name,
			TotalFetched:     provider.TotalFetched(),
			LastResponseTime: int64(provider.LastResponseTime().Milliseconds()),
			Enabled:          provider.Enabled(),
		})
	}
	return items, nil
}

type TorrentPageInfo struct {
	DownloadPath string            `json:"download_path"`
	MovieSize    int64             `json:"movie_size"`
	SeasonSize   int64             `json:"season_size"`
	Providers    []TorrentProvider `json:"providers"`
	Cloudflare   Cloudflare        `json:"cloudflare"`
}
type TorrentProvider struct {
	Name             string `json:"name"`
	TotalFetched     int64  `json:"total_fetched"`
	LastResponseTime int64  `json:"last_response_time"`
	Enabled          bool   `json:"enabled"`
}
type Cloudflare struct {
	ChallengeResolver  string `json:"challenge_resolver"`
	FlaresolverrUrl    string `json:"flaresolverr_url"`
	CapsolverrProxyUrl string `json:"capsolverr_proxy_url"`
	CapsolverrApiKey   string `json:"capsolverr_api_key"`
}
