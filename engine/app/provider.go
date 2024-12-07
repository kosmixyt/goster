package engine

import (
	"fmt"
	"sync"
)

var number_of_provider = 0

type TorrentProvider interface {
	Init() error
	Enabled() bool
	Search(Type string, query string, channel chan []*Torrent_File, wg *sync.WaitGroup)
}

var Providers []TorrentProvider = []TorrentProvider{&YGG, &SHAREWOOD}

func ProviderInit() {
	for _, provider := range Providers {
		fmt.Println("Provider: ", provider.Enabled())
		if provider.Enabled() {
			if err := provider.Init(); err != nil {
				panic(err)
			} else {
				number_of_provider++
			}
		}
	}
}

func Search(Type string, query string, withFormatter bool, channel chan []*Torrent_File, wg *sync.WaitGroup) {
	defer wg.Done()
	var deferedWaitGroup sync.WaitGroup
	proxyChan := make(chan []*Torrent_File, number_of_provider)
	if withFormatter {
		query = FormatTorrentNameSearch(query)
	}
	for _, provider := range Providers {
		if provider.Enabled() {
			deferedWaitGroup.Add(1)
			go provider.Search(Type, query, proxyChan, &deferedWaitGroup)
		}
	}
	deferedWaitGroup.Wait()
	var result []*Torrent_File
	for i := 0; i < number_of_provider; i++ {
		it := <-proxyChan
		result = append(result, it...)
	}
	close(proxyChan)
	channel <- result
}

func AddLatestTorrentTracker() {
	var files = make(byIndexCategory)
	if Config.TorrentProviders.YGG.Username != "" {
		files = YGG.FetchNewItems()
	}
	if Config.TorrentProviders.Sharewood.Username != "" {
		items := SHAREWOOD.FetchNewItems()
		for k, v := range items {
			if _, ok := files[k]; ok {
				files[k] = append(files[k], v...)
			} else {
				files[k] = v
			}
		}
	}
}
