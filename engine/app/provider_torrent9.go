package engine

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type Torrent9 struct {
}

var torrentnine = Torrent9{}

func (t *Torrent9) Init() error {
	return nil
}

func (t *Torrent9) Enabled() bool {
	return true
}

func (t *Torrent9) Search(Type string, query string, channel chan []*Torrent_File, wg *sync.WaitGroup) {
	defer wg.Done()
	torrents_file := []*Torrent_File{}

	url := "https://www.torrent9.cv/recherche/" + strings.ReplaceAll(url.QueryEscape(query), "+", "%20")
	fmt.Println(url, query)
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		panic(err)
	}
	table := doc.Find("table.table.table-striped.table-bordered.cust-table")
	table.Find("tr").Each(func(i int, s *goquery.Selection) {
		element := Torrent_File{}
		element.UUID = uuid.New().String()
		s.Find("td").Each(func(i int, s *goquery.Selection) {
			switch i {
			case 0:
				element.NAME = s.Find("a").Text()
				element.LINK, _ = s.Find("a").Attr("href")
				element.FetchData, _ = s.Find("a").Attr("href")
			case 1:
				element.SIZE_str = s.Text()
			case 2:
				element.SEED, err = strconv.Atoi(s.Text())
			case 3:
				element.LEECH, err = strconv.Atoi(s.Text())
			}
		})
		element.PROVIDER = "torrent9"
		element.LastFetch = time.Now()
		torrents_file = append(torrents_file, &element)
	})
	channel <- torrents_file
}

func (t *Torrent9) FetchTorrentFile(tf *Torrent_File) (io.Reader, error) {
	res, err := http.Get("https://www.torrent9.cv" + tf.FetchData)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		panic(err)
	}
	s := doc.Find("a.btn.btn-danger.download")
	if s == nil {
		return nil, errors.New("no download link found")
	}
	link, _ := s.Attr("href")
	res, err = http.Get("https://www.torrent9.cv" + link)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}
