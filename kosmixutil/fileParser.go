package kosmixutil

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

func GetType(f string, subPath string) string {
	e, _ := GetEpisode(f)
	if e == 0 {
		return "movie"
	}
	s, _ := GetSeason(f, subPath)
	if s == 0 {
		s, _ = GetSeason(f, subPath)
	}
	if s != 0 {
		return "episode"
	}
	return "movie"
}

func GetEpisode(f string) (int, string) {
	var episodeRegex = regexp.MustCompile(`((?:[ÉEeéPpIiSsOoDd]{7}(?:[ \t])|[Ee]{1})([0-9]{1,2}))`)
	var episode = episodeRegex.FindStringSubmatch(f)
	if len(episode) > 0 {
		data, err := strconv.Atoi(episode[2])
		if err != nil {
			return 0, ""
		}
		return data, episode[1]
	}
	return 0, ""
}
func GetSeason(f string, subPath string) (int, string) {
	var seasonRegex = regexp.MustCompile(`((?:[Ss]{1}|[SsAaIiSsOoNn]{6}[ \t])([0-9]{1,2}))`)
	for _, r := range []string{f, subPath} {
		var season = seasonRegex.FindStringSubmatch(r)
		if len(season) > 0 {
			data, err := strconv.Atoi(season[2])
			if err != nil {
				return 0, ""
			}
			return data, season[1]
		}
	}
	return 0, ""
}

func IsVideoFile(Filename string) bool {
	allowedExtension := []string{"mkv", "mp4", "avi", "webm"}
	fileExtension := strings.Split(Filename, ".")[len(strings.Split(Filename, "."))-1]
	for _, extension := range allowedExtension {
		if extension == fileExtension {
			return true
		}
	}
	return false
}
func GetYear(f string) int {
	var yearRegex, err = regexp2.Compile(`(?<=(?:.))(?:19|20)[0-9]{2}`, 0)
	if err != nil {
		panic(err)
	}
	match, err := yearRegex.FindStringMatch(f)
	if err != nil {
		panic(err)
	}
	if match != nil && match.GroupCount() > 0 {
		// return match.String()
		data, err := strconv.Atoi(match.String())
		if err != nil {
			return -1
		}
		return data
	}
	return -1
}

func GetFlags(f string) []string {
	fileName := f
	fileName = strings.ToLower(fileName)
	fileName = strings.ReplaceAll(fileName, "[", "")
	fileName = strings.ReplaceAll(fileName, "]", "")
	var predictNameRegex, err = regexp2.Compile(
		`[MULTImulti]{5}|(?:1080|720|2160)[pPiI]?|(?:web|dvd|br|hd|bd)[\-]?(?:rip|-dl|dl|[light]{4,5})|\.web\.|[HhxX]{0,1}[\.]?26[45]{1}(?:\-pop)?|atmos|avc(?=[\.\-\[])|vostfr|vost|(?<=[\[\.\- ])3d|avc|avi|mkv|extended|imax|(?<=[\.\-])light(?=[\.\-])|6ch|doc(?=[\.\-\[\]])|(?:[true]{1,4})?french|(?:hd|4k)light|(?:ac[\-]?3|eac3|aac)|(?:10|8){1,2}[. \-]{0,1}b[it]{0,2}[s]?|(?:(?<=[\.\-\[\{])(?:fr|en[g]?|es|nl|vo|nf)(?=[\.\-\]]))|vff|vfi|vfq|voa|vf[0-9]{1}|(?:blu[e]?ray)|(?<!(?:[0-9]))(?:(?:dd[p]?)?[57]{1}\.1)|dts|hevc|xvid|(?:m|pop|xs)?(?:hd)(?:gz|tv|cam|[r0-1plus]{4,7})?|extreme|unrated`, 0)

	if err != nil {
		panic(err)
	}

	match, err := predictNameRegex.FindStringMatch(fileName)
	if err != nil {
		panic(err)
	}

	if match != nil {
		return []string{match.String()}
	}
	return []string{}
}
func GetTitle(f string) string {
	fileName := f
	fileName = strings.ToLower(fileName)
	fileName = strings.ReplaceAll(fileName, "[", "")
	fileName = strings.ReplaceAll(fileName, "]", "")
	var flags = GetFlags(f)
	year := GetYear(f)
	if year != -1 {
		flags = append(flags, strconv.Itoa(year))
	}
	_, seasonData := GetSeason(f, "")
	if seasonData != "" {
		flags = append(flags, strings.ToLower(seasonData))
	}
	_, episodeData := GetEpisode(f)
	if episodeData != "" {
		flags = append(flags, strings.ToLower(episodeData))
	}
	var flagsIndexs = []int{}
	for i := 0; i < len(flags); i++ {
		flagsIndexs = append(flagsIndexs, strings.Index(strings.ToLower(fileName), flags[i]))
	}
	if len(flagsIndexs) == 0 {
		return ReturnGood(fileName)
	}
	var lowestIndex = flagsIndexs[0]
	for i := 0; i < len(flagsIndexs); i++ {
		if flagsIndexs[i] < lowestIndex {
			lowestIndex = flagsIndexs[i]
		}
	}
	if lowestIndex == -1 {
		return ReturnGood(fileName)
	}
	return ReturnGood(fileName[:lowestIndex])
}

var codec_expressions map[string]string = map[string]string{
	"H264": `[hx]264|avc`,
	"H265": `[hx]265|hevc`,
	"XVID": `xvid`,
}

func GetCodec(f string) string {
	f = strings.ToLower(f)
	for codec, expression := range codec_expressions {
		var codecRegex, err = regexp2.Compile(expression, 0)
		if err != nil {
			panic(err)
		}
		match, err := codecRegex.FindStringMatch(f)
		if err != nil {
			panic(err)
		}
		if match != nil {
			return codec
		}
	}
	return ""
}

var qualitys map[string]string = map[string]string{
	"1080p": "1080p|fullhd|fhd",
	"720p":  "720p|hd|720|1280",
	"2160p": "2160p|4k|uhd|2160|3840",
	"480p":  "480p|480",
	"360p":  "360p|360",
	"240p":  "240p|240",
}

func GetQuality(f string) string {
	f = strings.ToLower(f)
	for quality, expression := range qualitys {
		var qualityRegex, err = regexp2.Compile(expression, 0)
		if err != nil {
			panic(err)
		}
		match, err := qualityRegex.FindStringMatch(f)
		if err != nil {
			panic(err)
		}
		if match != nil {
			return quality
		}
	}
	return ""
}
func GetSource(f string) string {
	f = strings.ToLower(f)
	if strings.Contains(f, "bluray") {
		if strings.Contains(f, "remux") {
			return "blurayremux"
		}
		return "bluray"
	}
	if strings.Contains(f, "web") {
		if strings.Contains(f, "rip") {
			return "webrip"
		}
		if strings.Contains(f, "dl") {
			return "webdl"
		}
		return "web"
	}
	if strings.Contains(f, "dvd") {
		return "dvd"
	}
	if strings.Contains(f, "tv") {
		return "tv"
	}
	return ""
}
