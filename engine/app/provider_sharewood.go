package engine

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type Sharewood struct {
	Client *http.Client
	Token  string
}

type ShareWoodItem struct {
	Id              int64  `json:"id"`
	Info_hash       string `json:"info_hash"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Size            int64  `json:"size"`
	Leechers        int    `json:"leechers"`
	Seeders         int    `json:"seeders"`
	Times_completed int    `json:"times_completed"`
	Category_id     int    `json:"category_id"`
	Sub_category_id int    `json:"subcategory_id"`
	Language        string `json:"language"`
	Freeleech       int    `json:"free"`
	Double_up       int    `json:"doubleup"`
	Created_at      string `json:"created_at"`
}

var SHAREWOOD Sharewood = Sharewood{
	Client: &http.Client{},
	Token:  "",
}

func (s *Sharewood) FetchNewItems() byIndexCategory {
	return make(byIndexCategory)
}
func (s *Sharewood) Enabled() bool {
	return Config.TorrentProviders.Sharewood.Username != ""
}

func (s *Sharewood) Init() error {
	cookie, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	s.Client.Jar = cookie
	s.Login()
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			s.Login()
		}
	}()
	return nil
}
func ParseName(name string) string {
	name = strings.ReplaceAll(name, ":", " ")
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	return name
}

func (s *Sharewood) Login() {
	req, err := http.NewRequest("GET", "https://www.sharewood.tv/login", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.140 Safari/537.36")
	res, err := s.Client.Do(req)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		panic("Error logging in to sharewood")
	}
	s.Client.Jar.SetCookies(req.URL, res.Cookies())
	var strbody string
	body, _ := io.ReadAll(res.Body)
	strbody = string(body)
	if strings.Contains(strbody, "Just a moment...") {
		panic("Cloudflare detected")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strbody))
	if err != nil {
		panic(err)
	}
	var token = doc.Find(`form > input[name="_token"]`).First().AttrOr("value", "")
	fmt.Println("Token: ", token)
	args := "_token=" + token + "&username=" + Config.TorrentProviders.Sharewood.Username + "&password=" + Config.TorrentProviders.Sharewood.Password + "&submit="

	req, err = http.NewRequest("POST", "https://www.sharewood.tv/login", strings.NewReader(args))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.140 Safari/537.36")
	res, err = s.Client.Do(req)
	if err != nil {
		panic(err)
	}
	_, err = io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		fmt.Println("Error logging in to sharewood: ", res.Status)
		// fmt.Println(string(body))
		panic("Error logging in to sharewood")
	}
	fmt.Println("Logged in to sharewood")
	s.Token = token
	s.Client.Jar.SetCookies(req.URL, res.Cookies())
}
func (s *Sharewood) Search(Type string, query string, channel chan []*Torrent_File, wg *sync.WaitGroup) {
	defer wg.Done()
	encoded := url.QueryEscape(ParseName(query))
	url := "https://www.sharewood.tv/filterTorrents?_token=" + s.Token + "&search=" + encoded + "&description=&uploader=&tags=&sorting=created_at&direction=desc&qty=25&categories%5B%5D=1"
	req, err := http.NewRequest("GET", url, nil)
	fmt.Println("Searching ", url)
	if err != nil {
		panic(err)
	}
	res, err := s.Client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	var items []*Torrent_File
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		panic(err)
	}
	doc.Find("div.row.table-responsive-line").Each(func(i int, j *goquery.Selection) {
		var item = Torrent_File{}
		item.UUID = uuid.NewString()
		item.NAME = j.Find("div.titre-table > a.view-torrent").Text()
		Slug := j.Find("div.titre-table > a.view-torrent").AttrOr("data-slug", "")
		Id := j.Find("div.titre-table > a.view-torrent").AttrOr("data-id", "")
		// to replace
		seedStr := j.Find("div.bouton-s").First()
		item.SEED, err = strconv.Atoi(seedStr.Text())
		if err != nil {
			panic(err)
		}
		item.FetchData = "https://www.sharewood.tv/download/" + Slug + "." + Id
		item.PROVIDER = "sharewood"
		item.LastFetch = time.Now()
		items = append(items, &item)
	})
	channel <- items
}
func (s *Sharewood) FetchTorrentFile(item *Torrent_File) (io.Reader, error) {
	req, err := http.NewRequest("GET", item.FetchData, nil)
	if err != nil {
		panic(err)
	}
	res, err := s.Client.Do(req)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		fmt.Println("Error fetching torrent file from sharewood: ", res.Status)
		return nil, errors.New("error fetching torrent file from sharewood")
	}
	// fmt.Println("Fetched torrent file from sharewood")
	return res.Body, nil
}
