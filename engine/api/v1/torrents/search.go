package torrents

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

var ItemsTorrents map[string]*engine.Torrent_File = make(map[string]*engine.Torrent_File)

func SearchTorrentsController(db *gorm.DB, user *engine.User, query string, metadata string) (*[]TorrentItemRender, error) {
	if !user.ADMIN {
		return nil, fmt.Errorf("not allowed")
	}
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	fetchWithMetadata := metadata == "true"
	items := make(chan []*engine.Torrent_File, 1)
	var wgp sync.WaitGroup
	wgp.Add(1)
	go engine.Search("ns", query, false, items, &wgp)
	fmt.Println("waiting")
	wgp.Wait()
	fmt.Println("done")
	close(items)
	c := <-items
	v := make(chan *TorrentItemRender, len(c))
	var mwg sync.WaitGroup
	for _, item := range c {
		mwg.Add(1)
		ItemsTorrents[uuid.NewString()] = item
		go MapTorrentItem(item, &mwg, v, fetchWithMetadata)
	}
	mwg.Wait()
	var res []TorrentItemRender = make([]TorrentItemRender, 0)
	for i := 0; i < len(c); i++ {
		item := <-v
		if item != nil {
			res = append(res, *item)
		}
	}
	return &res, nil
}
func MapTorrentItem(item *engine.Torrent_File, wg *sync.WaitGroup, channel chan *TorrentItemRender, withMetadata bool) {
	defer wg.Done()
	escape := &TorrentItemRender{
		ProviderName: item.PROVIDER,
		Name:         item.NAME,
		Seed:         item.SEED,
		Link:         item.LINK,
		Id:           item.UUID,
	}
	if withMetadata {
		metadata, err := item.GetMetadata()
		if err != nil {
			fmt.Println("Error When getting metadata", err)
			channel <- nil
			return
		}
		wmtdt := JsonMetadata{
			Size: metadata.TotalLength(),
			Files: func() []FileItem {
				var res []FileItem
				for _, file := range metadata.UpvertedFiles() {
					res = append(res, FileItem{
						Path: filepath.Dir(file.DisplayPath(metadata)),
						Size: file.Length,
						Name: filepath.Base(file.DisplayPath(metadata)),
					})
				}
				return res
			}(),
		}

		// escape.Metadata = &wmtdt
		// fmt.Println("Size", wmtdt.Size, "Files", len(wmtdt.Files), item.NAME, item.FetchData, item.PATH)
		escape.Size = wmtdt.Size
	}
	channel <- escape
}

func SearchTorrents(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	query := ctx.Query("query")
	metadata := ctx.Query("metadata")
	d, err := SearchTorrentsController(db, &user, query, metadata)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, d)
}
func SearchTorrentsWs(db *gorm.DB, request *kosmixutil.WebsocketMessage, conn *websocket.Conn) {
	user, err := engine.GetUserWs(db, request.UserToken, []string{})
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	keys := kosmixutil.GetStringKeys([]string{"query", "metadata"}, request.Options)
	d, err := SearchTorrentsController(db, &user, keys["query"], keys["metadata"])
	if err != nil {
		kosmixutil.SendWebsocketResponse(conn, nil, err, request.RequestUuid)
		return
	}
	kosmixutil.SendWebsocketResponse(conn, d, nil, request.RequestUuid)
}

type TorrentItemRender struct {
	Id           string `json:"id"`
	ProviderName string `json:"provider_name"`
	Name         string `json:"name"`
	Link         string `json:"link"`
	Seed         int    `json:"seed"`
	Size         int64  `json:"size"`
	Flags        []string
}
type JsonMetadata struct {
	Size  int64      `json:"size"`
	Files []FileItem `json:"files"`
}
type FileItem struct {
	Size int64  `json:"size"`
	Name string `json:"name"`
	Path string `json:"path"`
}
