package admin

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	engine "kosmix.fr/streaming/engine/app"
	"kosmix.fr/streaming/kosmixutil"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func PtyConnector(ctx *gin.Context, db *gorm.DB) {
	user, err := engine.GetUser(db, ctx, []string{})
	if err != nil {
		ctx.JSON(401, gin.H{"error": "not logged in"})
		return
	}
	if !user.ADMIN {
		ctx.JSON(401, gin.H{"error": "not admin"})
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	refConn := &conn
	if err != nil {
		fmt.Println(err)
		return
	}
	c := exec.Command(kosmixutil.GetShell())
	c.Env = append(c.Env, "TERM=xterm-256color")
	f, err := pty.Start(c)
	if err != nil {
		(*refConn).WriteMessage(websocket.TextMessage, []byte(err.Error()))
		(*refConn).Close()
		return
	}

	closed := make(chan bool)
	go func() {
		for {
			var mess PtyMessage
			err := (*refConn).ReadJSON(&mess)
			if err != nil {
				fmt.Println("error reading websocket", err.Error())
				break
			}
			if mess.Resize != nil {
				fmt.Println("resize command")
				args := strings.Split(*mess.Resize, " ")
				if len(args) != 2 {
					fmt.Println("resize command not valid")
					continue
				}
				rowsS, colsS := args[0], args[1]
				row, err := strconv.Atoi(rowsS)
				if err != nil {
					fmt.Println("resize command not valid")
					continue
				}
				cols, err := strconv.Atoi(colsS)
				if err != nil {
					fmt.Println("resize command not valid")
					continue
				}
				fmt.Println("resizing pty to ", row, cols)
				pty.Setsize(f, &pty.Winsize{Rows: uint16(row), Cols: uint16(cols)})
				continue
			}
			f.Write([]byte(mess.Command))
		}
		closed <- true
	}()
	go func() {
		for {
			outputBuffer := make([]byte, 1024)
			n, err := f.Read(outputBuffer)
			if err != nil {
				fmt.Println("error reading pty", err.Error())
				break
			}
			(*refConn).WriteMessage(websocket.TextMessage, outputBuffer[:n])
		}
		closed <- true
	}()
	<-closed
	f.Close()
	fmt.Println("closing pty")
	conn.Close()
	f = nil

}

type PtyMessage struct {
	Command string  `json:"c"`
	Resize  *string `json:"r"`
}
