package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/gorilla/websocket"
	"github.com/kpango/glg"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serveWsStatus(w http.ResponseWriter, r *http.Request) {
	serveWs(w, r, globalstate.Instance())
}

func serveWs(w http.ResponseWriter, r *http.Request, in interface{}) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glg.Errorf("error while upgrading connection to websocket: %s", err)
	}
	glg.Infof("client %s connected to status websocket", r.RemoteAddr)
	oniiChan := make(chan string)
	go readData(ws, oniiChan)
	go pushData(ws, in, oniiChan)
}

func readData(ws *websocket.Conn, oniiChan chan string) {
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				glg.Infof("client %s disconnected from websocket", ws.RemoteAddr())
			} else {
				glg.Warnf("error while writing to websocket: %s", err)
			}
			oniiChan <- "STOP"
			return
		}
		glg.Debugf("received message: %s", message)
	}
}

func pushData(ws *websocket.Conn, in interface{}, oniiChan chan string) {
	defer ws.Close()
	for {
		select {
		case msg := <-oniiChan:
			if msg == "STOP" {
				glg.Debugf("stopping websocket push")
				return
			}
		default:
		}
		out, err := encodeJson(in)
		if err != nil {
			glg.Warnf("error while marshaling encoder to json: %s", err)
			break
		}
		err = ws.WriteMessage(websocket.TextMessage, out)
		if err != nil {
			glg.Errorf("error while writing to websocket: %s", err)
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func encodeJson(in interface{}) ([]byte, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(in)
	if err != nil {
		return nil, err
	}
	err = writer.Flush()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
