package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
)

const (
	stdin  = 0
	stdout = 1
	stderr = 2
)

func main() {
	portNo := flag.Int("port", 0, "Port to listen for websocket connections")
	ipAddress := flag.String("ipAddress", "", "SSH Server IP Address")
	username := flag.String("username", "", "SSH User")
	password := flag.String("password", "", "SSH Password")
	flag.Parse()
	if err := checkInputs(*portNo, *ipAddress, *username, *password); err != nil {
		log.Fatal(err)
	}
	server := http.Server{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		Addr: fmt.Sprintf(":%d", *portNo),
		Handler: &WsHandler{
			username:  *username,
			password:  *password,
			ipAddress: *ipAddress,
		},
	}
	log.Fatal(server.ListenAndServe())
}

func checkInputs(portNo int, ipAddress, username, password string) error {
	if portNo == 0 {
		return errors.New("port no is mandatory")
	}
	if ipAddress == "" {
		return errors.New("IP Address is mandatory")
	}
	if username == "" {
		return errors.New("username is mandatory")
	}
	if password == "" {
		return errors.New("password is mandatory")
	}
	return nil
}

type WsHandler struct {
	username     string
	password     string
	ipAddress    string
	outputWriter io.Writer
	errorWriter  io.Writer
	inputReader  *InputReader
}

type OutputWriter struct {
	conn *websocket.Conn
}

func (e *OutputWriter) Write(p []byte) (int, error) {
	d := append([]byte{stdout}, p...)
	err := e.conn.WriteMessage(websocket.BinaryMessage, d)
	if err != nil {
		return 0, err
	}
	return len(p), err
}

type ErrorWriter struct {
	conn *websocket.Conn
}

func (e *ErrorWriter) Write(p []byte) (int, error) {
	d := append([]byte{stderr}, p...)
	err := e.conn.WriteMessage(websocket.BinaryMessage, d)
	if err != nil {
		return 0, err
	}
	return len(p), err
}

type InputReader struct {
	conn   *websocket.Conn
	writer io.WriteCloser
}

func (i *InputReader) Stream() {
	for {
		t, p, err := i.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		if t == websocket.BinaryMessage {
			if p[0] == stdin {
				_, err := i.writer.Write(p[1:])
				if err != nil {
					log.Println(err)
					break
				}
			}
		}
	}
}

func (ws *WsHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	ws.connectToSsh(writer, req)
}

func (ws *WsHandler) connectToSsh(writer http.ResponseWriter, req *http.Request) {
	upgrader := NewUpgrader()
	conn, err := upgrader.Upgrade(writer, req, nil)
	if err != nil {
		log.Fatal(err)
	}
	ws.errorWriter = &ErrorWriter{conn: conn}
	ws.outputWriter = &OutputWriter{conn: conn}
	client := NewSshClient(ws.username, ws.password, ws.ipAddress, 22)
	pipeWriter, err := client.Connect(ws.outputWriter, ws.errorWriter)
	if err != nil {
		log.Println(err)
	}
	ws.inputReader = &InputReader{
		conn:   conn,
		writer: pipeWriter,
	}
	go ws.inputReader.Stream()
}

// NewUpgrader - Creates a new websocket upgrader
func NewUpgrader() websocket.Upgrader {
	upgrader := websocket.Upgrader{}
	upgrader.Subprotocols = []string{"binary"}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	return upgrader
}
