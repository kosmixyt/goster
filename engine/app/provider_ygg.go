package engine

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
	"kosmix.fr/streaming/kosmixutil"
)

const YGG_URL = "https://www.ygg.re"

var MovieCategoryYgg []int = []int{2183}
var TvCategoryYgg []int = []int{2184}

var UrlToDownloadId = regexp2.MustCompile(`\/([0-9]{3,})`, 0)
var last_response_time time.Duration = time.Duration(0) * time.Second
var total_fetched int = 0

type Flaresolverr struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		UserAgent string   `json:"userAgent"`
		Url       string   `json:"url"`
		Status    int      `json:"status"`
		Cookies   []cookie `json:"cookies"`
		Headers   struct {
		} `json:"headers"`
		Response string `json:"response"`
	} `json:"solution"`
	StartTimestamp int64  `json:"startTimestamp"`
	EndTimestamp   int64  `json:"endTimestamp"`
	Version        string `json:"version"`
}
type cookie struct {
	Domain   string `json:"domain"`
	Expires  int    `json:"expiry"`
	HttpOnly bool   `json:"httpOnly"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	SameSite string `json:"sameSite"`
	Secure   bool   `json:"secure"`
	Value    string `json:"value"`
}

type CapSolverrCreateTaskPayload struct {
	ClientKey string                          `json:"clientKey"`
	Task      CapSolverrCreateTaskPayloadTask `json:"task"`
}
type CapSolverrCreateTaskPayloadTask struct {
	Type       string `json:"type"`
	WebsiteURL string `json:"websiteURL"`
	Proxy      string `json:"proxy"`
}
type CapSolverrCreateTaskResponse struct {
	Status  string `json:"status"`
	ErrorId int    `json:"errorId"`
	TaskId  string `json:"taskId"`
}
type CapSolverrResultTaskPayload struct {
	ClientKey string `json:"clientKey"`
	TaskId    string `json:"taskId"`
}
type CapSolverrResultTaskResponse struct {
	Status   string `json:"status"`
	ErrorId  int    `json:"errorId"`
	TaskId   string `json:"taskId"`
	Solution struct {
		Cookies []map[string]string `json:"cookies"`
	} `json:"solution"`
	Proxy     string `json:"proxy"`
	Token     string `json:"token"`
	Type      string `json:"type"`
	UserAgent string `json:"userAgent"`
}

var dialer *net.Dialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	LocalAddr: nil,
	DualStack: false, // This forces the use of IPv4
}

var YGG YggTorrent = YggTorrent{
	Cookie:      "",
	connected:   false,
	client:      nil,
	credentials: map[string]string{},
}

func (y *YggTorrent) TotalFetched() int64 {
	return int64(total_fetched)
}
func (y *YggTorrent) LastResponseTime() time.Duration {
	return last_response_time
}
func (y *YggTorrent) Enabled() bool {
	return y.credentials["username"] != "" && y.credentials["password"] != "" && y.connected
}
func (y *YggTorrent) Name() string {
	return "ygg"
}
func (y *YggTorrent) Test() error {
	if y.Cookie == "" {
		return errors.New("Not connected")
	}
	channel_torrents := make(chan []*Torrent_File)
	var wg sync.WaitGroup
	wg.Add(1)
	go y.Search("movie", "test", channel_torrents, &wg)
	wg.Wait()
	if len(<-channel_torrents) == 0 {
		return errors.New("Failed to get torrents")
	}
	return nil
}

func ParseProxyUrl() *url.URL {
	elements := strings.Split(Config.Cloudflare.CapsolverrProxyUrl, ":")
	//host:port:username:password
	var proxyUrl string
	if len(elements) == 4 {
		proxyUrl = "http://" + elements[2] + ":" + elements[3] + "@" + elements[0] + ":" + elements[1]
	} else {
		panic("Invalid proxy url")
	}
	proxy, err := url.Parse(proxyUrl)
	if err != nil {
		panic(err)
	}
	return proxy
}

func InitHttpClient() error {
	challenge_type := Config.Cloudflare.ChallengeResolver
	if challenge_type == "flaresolverr" {
		if Config.Cloudflare.FlaresolverrUrl == "" {
			panic("Challenge type is flaresolverr but no url found")
		}
		fmt.Println("Using flaresolverr")
		YGG.client = &http.Client{
			Timeout: 100 * time.Second,
			Transport: &http.Transport{
				Dial: dialer.Dial,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		return nil
	}
	if challenge_type == "capsolverr" {
		if Config.Cloudflare.CapsolverrApiKey == "" || Config.Cloudflare.CapsolverrProxyUrl == "" {
			return errors.New("challenge type is capsolverr but no api key or proxy url found")
		}
		fmt.Println("Using capsolverr")
		YGG.client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(ParseProxyUrl()),
				Dial:  dialer.Dial,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		return nil
	}
	if challenge_type == "puppeteer" {
		if _, err := os.Stat(kosmixutil.GetWrapperPath()); os.IsNotExist(err) {
			return errors.New("wrapper not found")
		}
		fmt.Println("Using puppeteer")
		YGG.client = &http.Client{}
		return nil
	}
	return errors.New("invalid challenge type :" + challenge_type)
}

func (y *YggTorrent) TryEnable(credentials map[string]string) error {
	var err error
	if err := InitHttpClient(); err != nil {
		return err
	}
	YGG.UserAgent, YGG.Cookie, err = YGG.GetClearance()
	if err != nil {
		return err
	}
	if credentials["username"] == "" || credentials["password"] == "" {
		return errors.New("credentials not found")
	}
	if success, err := YGG.Login(credentials); !success {
		fmt.Println("Failed to connect to ygg torrent [first attempt]", err.Error())
		return err
	}
	y.connected = true
	y.credentials = credentials
	go func() {
		a := 0
		for {
			time.Sleep(30 * time.Minute)
			YGG.RefreshCookie()
			a += 1
			if success, err := y.Login(y.credentials); !success {
				y.connected = false
				fmt.Println("Failed to connect to ygg torrent", err.Error(), "attemp : ", a)
			} else {
				y.connected = true
				fmt.Println("Success to refresh YGG cookies after 30 minutes")
			}

		}
	}()
	return nil
}

type ReturnStructJs struct {
	UserAgent string `json:"userAgent"`
	Cookies   string `json:"cookies"`
}

func (y *YggTorrent) GetClearance() (string, string, error) {
	challenge_type := Config.Cloudflare.ChallengeResolver
	if challenge_type == "flaresolverr" {
		method := "POST"
		payload := strings.NewReader(`{"cmd":"request.get", "url" : "` + YGG_URL + `/", "timeout":160000}`)
		req, err := http.NewRequest(method, Config.Cloudflare.FlaresolverrUrl, payload)
		if err != nil {
			return "", "", err
		}
		req.Header.Add("Content-Type", "application/json")
		res, err := YGG.client.Do(req)
		if err != nil {
			return "", "", err
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return "", "", err
		}
		el := Flaresolverr{}
		er := json.Unmarshal(body, &el)
		if er != nil {
			return "", "", er
		}
		var cookie string
		for _, v := range el.Solution.Cookies {
			cookie = v.Name + "=" + v.Value + "; " + cookie
		}
		return el.Solution.UserAgent, cookie, nil
	}
	if challenge_type == "puppeteer" {
		args := []string{"ygg"}
		cmd := exec.Command("./wrapper", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return "", "", err
		}
		if err := cmd.Start(); err != nil {
			return "", "", err
		}

		fmt.Println("Waiting for puppeteer to solve the challenge")
		go io.Copy(os.Stderr, stderr)
		fmt.Println("Puppeteer solved the challenge")
		out, err := io.ReadAll(stdout)
		if err != nil {
			return "", "", err
		}
		fmt.Println(string(out))
		var el ReturnStructJs
		er := json.Unmarshal(out, &el)
		if er != nil {
			return "", "", er
		}
		return el.UserAgent, el.Cookies, nil
	}
	if challenge_type == "capsolverr" {
		payload := CapSolverrCreateTaskPayload{
			ClientKey: Config.Cloudflare.CapsolverrApiKey,
			Task: CapSolverrCreateTaskPayloadTask{
				Type:       "AntiCloudflareTask",
				WebsiteURL: YGG_URL,
				Proxy:      ParseProxyUrl().String(),
			},
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			panic(err)
		}
		body := bytes.NewReader(payloadBytes)
		req, err := http.NewRequest("POST", "https://api.capsolver.com/createTask", body)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := YGG.client.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		resbody, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		el := CapSolverrCreateTaskResponse{}
		er := json.Unmarshal(resbody, &el)
		if er != nil {
			panic(er)
		}
		TaskId := el.TaskId
		if el.ErrorId != 0 {
			fmt.Println(string(resbody))
			fmt.Println(el.Status, el.TaskId)
			panic("Failed to create task" + el.Status)
		} else {
			fmt.Println("Task created with id : ", TaskId)
		}
		for {
			payload := CapSolverrResultTaskPayload{
				ClientKey: Config.Cloudflare.CapsolverrApiKey,
				TaskId:    TaskId,
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				panic(err)
			}
			body := bytes.NewReader(payloadBytes)
			req, err := http.NewRequest("POST", "https://api.capsolver.com/getTaskResult", body)
			if err != nil {
				panic(err)
			}
			req.Header.Set("Content-Type", "application/json")
			res, err := YGG.client.Do(req)
			if err != nil {
				panic(err)
			}
			defer res.Body.Close()
			resbody, err := io.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}
			el := CapSolverrResultTaskResponse{}
			er := json.Unmarshal(resbody, &el)
			if er != nil {
				panic(er)
			}
			if el.Status == "success" {
				fmt.Println("Success to get clearance")
				return el.UserAgent, el.Solution.Cookies[0]["name"] + "=" + el.Solution.Cookies[0]["value"], nil
			}
			fmt.Println("Waiting for capsolverr to solve the challenge")
			time.Sleep(2 * time.Second)
		}

	}
	panic("Invalid challenge type " + challenge_type)
}
func (y *YggTorrent) Login(credentials map[string]string) (bool, error) {
	url := YGG_URL + "/auth/process_login"
	method := "POST"
	body := &bytes.Buffer{}
	formData := multipart.NewWriter(body)
	formData.WriteField("id", credentials["username"])
	formData.WriteField("pass", credentials["password"])
	formData.WriteField("ci_csrf_token", "")
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", y.UserAgent)
	req.Header.Add("Cookie", y.Cookie)
	req.Header.Set("Content-Type", formData.FormDataContentType())
	fmt.Println(url)
	var res *http.Response
	res, err = y.client.Do(req)
	if err != nil || res.StatusCode != 200 {
		y.connected = false
		return false, errors.New("Failed To login to ygg " + strconv.Itoa(res.StatusCode) + "; isCloudflare ? " + strconv.FormatBool(y.isCloudFlareRejects(res.Body)))
	}
	if y == nil {
		panic("nil pointer to ygg")
	}
	y.Cookie = res.Header.Get("Set-Cookie") + ";" + y.Cookie
	y.connected = true
	fmt.Println("Connected to ygg")
	return true, nil
}
func (y *YggTorrent) isCloudFlareRejects(body io.Reader) bool {
	strbody, _ := io.ReadAll(body)
	return strings.Contains(string(strbody), "Just a moment...")

}

type YggTorrent struct {
	Cookie      string
	client      *http.Client
	UserAgent   string
	connected   bool
	credentials map[string]string
}

func GetSearchUrl(Type string, name string) string {
	switch Type {
	case "movie":
		return YGG_URL + "/engine/search?name=" + name + "&category=2145&sub_category=all&do=search&order=desc&sort=seed"
	case "tv":
		return YGG_URL + "/engine/search?name=" + name + "&category=2145&sub_category=all&do=search&order=desc&sort=seed"
	case "ns":
		fmt.Println("[WARN] ns not recommended")
		return YGG_URL + "/engine/search?name=" + name + "&category=2145&sub_category=all&do=search&order=desc&sort=seed"
	}

	panic("Type not found")
}
func (y *YggTorrent) RefreshCookie() {

	var err error
	y.UserAgent, y.Cookie, err = y.GetClearance()
	if err != nil {
		panic(err)
	}
}

func FormatTorrentNameSearch(name string) string {
	name = strings.ReplaceAll(name, " ", ".")
	name = strings.ReplaceAll(name, ":", "")
	name = strings.ReplaceAll(name, "'", "")
	return name
}

var index = 0

func (y *YggTorrent) Search(Type string, query string, channel chan [](*Torrent_File), wg *sync.WaitGroup) {
	start := time.Now()
	defer wg.Done()
	if y.Cookie == "" {
		channel <- []*Torrent_File{}
		return
	}
	strbody := ""
	url := GetSearchUrl(Type, query)
	req, err := http.NewRequest("GET", url, nil)
	fmt.Println("Search", url)
	if err != nil {
		fmt.Println(err)
		channel <- []*Torrent_File{}
		return
	}
	req.Header.Add("User-Agent", y.UserAgent)
	req.Header.Add("Cookie", y.Cookie)
	res, _ := (y.client.Do(req))
	if res == nil {
		panic(res)
	}
	body, _ := io.ReadAll(res.Body)
	strbody = string(body)
	if strings.Contains(strbody, "Just a moment...") {
		fmt.Println("Cloudflare detected")
		channel <- []*Torrent_File{}
		return
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strbody))
	if err != nil {
		panic(err)
	}
	var torrents []*Torrent_File
	haveBody := false
	doc.Find("tbody").Each(func(i int, s *goquery.Selection) {
		if i != 1 {
			return
		}
		haveBody = true
		fmt.Println("Page Have the good Html Table")
		s.Find("tr").Each(func(j int, l *goquery.Selection) {
			element := Torrent_File{}
			element.UUID = uuid.NewString()
			l.Find("td").Each(func(k int, m *goquery.Selection) {
				switch k {
				case 1:
					element.NAME = m.Find("a").Text()
					element.LINK, _ = m.Find("a").Attr("href")
				case 2:
					element.FetchData, _ = m.Find("a").Attr("target")
				case 7:
					seedInt, err := strconv.Atoi(m.Text())
					if err != nil {
						panic(err)
					}
					element.SEED = seedInt
				}
			})
			element.PROVIDER = y.Name()
			element.LastFetch = time.Now()
			torrents = append(torrents, &element)

		})
	})

	if !haveBody && !strings.Contains(strbody, "Aucun résultat ! Essayez d'élargir votre recherche...") {
		fmt.Println("No body found")
		index += 1
		WriteFile(strings.NewReader(strbody), "ygg-"+strconv.Itoa(index)+".html")
		// panic("No body found")
	}
	fmt.Println("Time to fetch ygg : ", time.Since(start))
	last_response_time = time.Since(start)
	channel <- torrents
}
func (y *YggTorrent) FetchTorrentFile(t *Torrent_File) (io.Reader, error) {
	total_fetched = total_fetched + 1
	if !y.connected {
		if success, _ := y.Login(y.credentials); !success {
			panic("Failed to Login to ygg torrent")
		}
	}
	id := strings.ReplaceAll(t.FetchData, " ", "")
	req, err := http.NewRequest("GET", YGG_URL+"/engine/download_torrent?id="+id, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", y.UserAgent)
	req.Header.Add("Cookie", y.Cookie)
	res, _ := y.client.Do(req)
	if res.StatusCode != 200 {
		return nil, errors.New("failed to get torrent file")
	}
	return res.Body, nil
}
func WriteToFile(data string, path string) {
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(data))
}

type AjaxResponse = map[string][]interface{}
type byIndexCategory map[uint][]*Torrent_File

func (y *YggTorrent) FetchNewItems() byIndexCategory {
	if !y.connected {
		if success, _ := y.Login(y.credentials); !success {
			panic("Failed to Login to ygg torrent")
		}
	}
	req, err := http.NewRequest("GET", YGG_URL+"/engine/ajax_top_query/day", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", y.UserAgent)
	req.Header.Add("Cookie", y.Cookie)
	res, _ := y.client.Do(req)
	if res.StatusCode != 200 {
		panic("Failed to get new items")
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var el AjaxResponse
	er := json.Unmarshal(body, &el)
	if er != nil {
		fmt.Println(string(body))
		panic(er)
	}

	var NewTorrentsItems = make(byIndexCategory)
	for _, v := range append(MovieCategoryYgg, TvCategoryYgg...) {
		items := el[strconv.Itoa(v)]
		for _, item := range items {
			torrentItemFromlink := Torrent_File{}
			tableau, ok := item.([]interface{})
			if !ok {
				// fmt.Println(item, " is a nombre ?")
				continue
			}
			link := (tableau[1].(string))
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(link))
			if err != nil {
				panic(err)
			}
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				if i == 0 {
					val, exist := s.Attr("href")
					if !exist {
						panic("No href found")
					}
					matched, err := UrlToDownloadId.FindStringMatch(val)
					if err != nil {
						panic(err)
					}
					if matched == nil {
						panic("No match found" + val)
					}
					torrentItemFromlink.FetchData = matched.Captures[0].String()[1:]
					torrentItemFromlink.LINK = val
					torrentItemFromlink.NAME = s.Text()
				}
			})
			torrentItemFromlink.PROVIDER = y.Name()
			intSeed, err := strconv.Atoi(tableau[7].(string))
			if err != nil {
				panic(err)
			}
			intLeech, err := strconv.Atoi(tableau[3].(string))
			if err != nil {
				panic(err)
			}
			torrentItemFromlink.SEED = intSeed
			torrentItemFromlink.LEECH = intLeech
			// NewTorrentsItems = append(NewTorrentsItems, &torrentItemFromlink)
			if _, ok := NewTorrentsItems[uint(v)]; !ok {
				NewTorrentsItems[uint(v)] = make([]*Torrent_File, 0)
			}
			NewTorrentsItems[uint(v)] = append(NewTorrentsItems[uint(v)], &torrentItemFromlink)
		}
	}
	return NewTorrentsItems
}
