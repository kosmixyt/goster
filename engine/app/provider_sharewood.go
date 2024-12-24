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
	Client      *http.Client
	Token       string
	credentials map[string]string
}

var s_last_response_time time.Duration = time.Duration(0) * time.Millisecond
var s_total_fetched int64 = 0
var base_url = "https://www.sharewood.tv"

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
	Client:      &http.Client{},
	Token:       "",
	credentials: map[string]string{},
}

func (s *Sharewood) Name() string {
	return "sharewood"
}
func (s *Sharewood) TotalFetched() int64 {
	return s_total_fetched
}
func (s *Sharewood) LastResponseTime() time.Duration {
	return s_last_response_time
}

func (s *Sharewood) FetchNewItems() byIndexCategory {
	return make(byIndexCategory)
}
func (s *Sharewood) Enabled() bool {
	return s.credentials["username"] != "" && s.credentials["password"] != "" && s.Token != ""
}
func (s *Sharewood) Test() error {
	items := make(chan []*Torrent_File)
	var wg sync.WaitGroup
	wg.Add(1)
	go s.Search("movies", "interstellar", items, &wg)
	wg.Wait()
	if len(<-items) == 0 {
		return errors.New("No items found")
	}
	return nil
}

func (s *Sharewood) TryEnable(credentials map[string]string) error {
	cookie, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	s.Client.Jar = cookie
	if err := s.Login(credentials); err != nil {
		return err

	}
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			s.Login(s.credentials)
		}
	}()
	return nil
}
func ParseName(name string) string {
	name = strings.ReplaceAll(name, ":", " ")
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	return name
}

func (s *Sharewood) Login(credentials map[string]string) error {
	if credentials["username"] == "" || credentials["password"] == "" {
		return errors.New("Missing credentials")
	}
	req, err := http.NewRequest("GET", base_url+"/login", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.140 Safari/537.36")
	res, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("Error getting login page from sharewood, status: " + res.Status)
	}
	s.Client.Jar.SetCookies(req.URL, res.Cookies())
	var strbody string
	body, _ := io.ReadAll(res.Body)
	strbody = string(body)
	if strings.Contains(strbody, "Just a moment...") {
		return errors.New("Cloudflare detected")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strbody))
	if err != nil {
		panic(err)
	}
	var token = doc.Find(`form > input[name="_token"]`).First().AttrOr("value", "")
	fmt.Println("Token: ", token)
	args := "_token=" + token + "&username=" + credentials["username"] + "&password=" + credentials["password"] + "&submit="

	req, err = http.NewRequest("POST", base_url+"/login", strings.NewReader(args))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.140 Safari/537.36")
	res, err = s.Client.Do(req)
	if err != nil {
		return err
	}
	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		fmt.Println("Error logging in to sharewood: ", res.Status)
		// fmt.Println(string(body))
		return errors.New("Error logging in to sharewood")
	}
	fmt.Println("Logged in to sharewood")
	s.Token = token
	s.Client.Jar.SetCookies(req.URL, res.Cookies())
	return nil
}
func (s *Sharewood) Search(Type string, query string, channel chan []*Torrent_File, wg *sync.WaitGroup) {
	start := time.Now()
	defer wg.Done()
	encoded := url.QueryEscape(ParseName(query))
	url := base_url + "/filterTorrents?_token=" + s.Token + "&search=" + encoded + "&description=&uploader=&tags=&sorting=created_at&direction=desc&qty=25&categories%5B%5D=1"
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
		item.FetchData = base_url + "/download/" + Slug + "." + Id
		item.PROVIDER = s.Name()
		item.LastFetch = time.Now()
		items = append(items, &item)
	})
	s_last_response_time = time.Since(start)
	channel <- items
}
func (s *Sharewood) FetchTorrentFile(item *Torrent_File) (io.Reader, error) {
	s_total_fetched++
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
