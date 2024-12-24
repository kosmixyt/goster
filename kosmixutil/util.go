package kosmixutil

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const PREDICT_NAME_SEPARATOR = "{@@@}"

func SendEvent(ctx *gin.Context, event string, data string) {
	event = strings.ReplaceAll(event, "\n", "")
	data = strings.ReplaceAll(data, "\n", "")
	select {
	case <-ctx.Request.Context().Done():
		return
	default:
		fmt.Fprintf(ctx.Writer, "event: %s\n", event)
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", data)
		ctx.Writer.Flush()
	}
}

func SendWebsocketResponse(websocket *websocket.Conn, data interface{}, err error, reqId string) {
	res := WebsocketResponse{
		RequestUuid: reqId,
		Data:        data,
		Error:       "",
	}
	if err != nil {
		res.Error = err.Error()
	}
	bdata, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	err = websocket.WriteMessage(1, bdata)
	if err != nil {
		panic(err)
	}
}

type WebsocketMessage struct {
	RequestUuid string      `json:"requestUuid"`
	Type        string      `json:"type"`
	Options     interface{} `json:"options"`
	UserToken   string      `json:"userToken"`
}
type WebsocketResponse struct {
	RequestUuid string      `json:"requestUuid"`
	Data        interface{} `json:"data"`
	Error       string      `json:"error"`
}

func ReturnGood(name string) string {
	re := regexp2.MustCompile(`\[[A-Za-z0-9]*\]`, 0)
	name, _ = re.Replace(name, " ", -1, -1)
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "  ", " ")
	name = strings.ReplaceAll(name, "  ", " ")
	name = strings.ReplaceAll(name, "  ", " ")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, "fr en", "")
	return name
}

type Range struct {
	Start  int64
	Length int64
}

// ContentRange returns Content-Range header value.
func (r Range) ContentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.Start, r.Start+r.Length-1, size)
}

var (
	// ErrNoOverlap is returned by ParseRange if first-byte-pos of
	// all of the byte-range-spec values is greater than the content size.
	ErrNoOverlap = errors.New("invalid range: failed to overlap")

	// ErrInvalid is returned by ParseRange on invalid input.
	ErrInvalid = errors.New("invalid range")
)

// ParseRange parses a Range header string as per RFC 7233.
// ErrNoOverlap is returned if none of the ranges overlap.
// ErrInvalid is returned if s is invalid range.
func ParseRange(s string, size int64) ([]Range, error) { // nolint:gocognit
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, ErrInvalid
	}
	var ranges []Range
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, ErrInvalid
		}
		start, end := textproto.TrimString(ra[:i]), textproto.TrimString(ra[i+1:])
		var r Range
		if start == "" {
			if end == "" || end[0] == '-' {
				return nil, ErrInvalid
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return nil, ErrInvalid
			}
			if i > size {
				i = size
			}
			r.Start = size - i
			r.Length = size - r.Start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, ErrInvalid
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.Start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.Length = size - r.Start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.Start > i {
					return nil, ErrInvalid
				}
				if i >= size {
					i = size - 1
				}
				r.Length = i - r.Start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, ErrNoOverlap
	}
	return ranges, nil
}

func ServerRangeRequest(ctx *gin.Context, size int64, reader io.ReadSeekCloser, supportMulti bool, norangesupported bool) {
	ctx.Writer.Header().Set("Accept-Ranges", "bytes")
	ranges, err := ParseRange(ctx.GetHeader("Range"), size)
	if err != nil {
		ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
		ctx.AbortWithStatus(416)
		return
	}
	if !norangesupported && len(ranges) == 0 {
		ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
		ctx.AbortWithStatus(416)
		return
	}
	if len(ranges) <= 0 {
		ctx.Status(200)
		ctx.Header("Content-Length", strconv.FormatInt(size, 10))
		ctx.Writer.WriteHeaderNow()
		io.Copy(ctx.Writer, reader)
		reader.Close()
		return
	}
	if len(ranges) == 1 {
		ctx.Status(206)
		ctx.Header("Content-Length", strconv.FormatInt(ranges[0].Length, 10))
		ctx.Header("Content-Range", ranges[0].ContentRange(size))
		ctx.Writer.WriteHeaderNow()
		reader.Seek(ranges[0].Start, io.SeekStart)
		io.CopyN(ctx.Writer, reader, ranges[0].Length)
		reader.Close()
		return
	}
	if !supportMulti {
		ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
		ctx.AbortWithStatus(416)
		return
	}
	ctx.Status(206)
	boundaryName := "foo"
	ctx.Header("Content-Type", "multipart/byteranges; boundary="+boundaryName)
	ctx.Writer.WriteHeaderNow()
	for _, r := range ranges {
		ctx.Header("Content-Length", strconv.FormatInt(r.Length, 10))
		ctx.Header("Content-Range", r.ContentRange(size))
		ctx.Writer.WriteHeaderNow()
		reader.Seek(r.Start, io.SeekStart)
		io.CopyN(ctx.Writer, reader, r.Length)
		ctx.Writer.Write([]byte("\r\n--" + boundaryName + "\r\n"))
	}
	reader.Close()

}

//	func ServerNonSeekable(ctx *gin.Context, reader io.ReadCloser) {
//		ctx.Status(200)
//		ctx.Writer.WriteHeaderNow()
//		for {
//			buf := make([]byte, 1024)
//			n, err := reader.Read(buf)
//			if err != nil {
//				if err == io.EOF {
//					fmt.Println("EOF")
//					break
//				}
//				panic(err)
//			}
//			_, err = ctx.Writer.Write(buf[:n])
//			if err != nil {
//				panic(err)
//			}
//			ctx.Writer.Flush()
//		}
//		err := reader.Close()
//		if err != nil {
//			panic(err)
//		}
//	}
func FormatFilenameForContentDisposition(filename string) string {
	// Remplacer les caractères interdits par des tirets bas
	filename = strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r == '/' || r == '*' || r == '?' || r == ':' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, filename)

	// Encoder les caractères spéciaux avec url.QueryEscape
	filename = url.QueryEscape(filename)

	return filename
}

func GenerateRandomKey(size int) string {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func GetStringKey(key string, opt interface{}) string {
	val, exist := opt.(map[string]interface{})[key].(string)
	if !exist {
		return ""
	}
	return val
}
func GetStringKeys(keys []string, opt interface{}) map[string]string {
	val := make(map[string]string)
	for _, key := range keys {
		val[key] = GetStringKey(key, opt)
	}
	return val
}

type PathElement struct {
	Path string `json:"path"`
	// size allowed to use for
	Size int64 `json:"size"`
}
