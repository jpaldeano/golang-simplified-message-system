package test

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	msgSystemHub "github.com/jpaldi/golang-simplified-message-system/server"
)

func TestGetID(t *testing.T) {
	var clientX *TestClient
	address := "localhost:8080"
	go msgSystemHub.InitHub(address)
	clientX = newTestClient(address)

	go clientX.assertGetIDResponseFromServer(t)
	go clientX.WS.WriteMessage(1, []byte("id"))
	time.Sleep(time.Second * 2) // breathing room to receive the server response
}

type TestClient struct {
	WS   *websocket.Conn
	Data chan []byte
}

func newTestClient(address string) *TestClient {
	var addr = flag.String("addr", address, "http service address")

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	client := &TestClient{WS: c, Data: make(chan []byte)}
	go client.read()

	return client

}

func (c *TestClient) assertGetIDResponseFromServer(t *testing.T) {
	for {
		select {
		case msg := <-c.Data:
			if !strings.HasPrefix(string(msg), "server: ") {
				t.Fatalf("unexpected response from server: expected to be prefixed by 'server: ', got %s", string(msg))
			}

			portString := strings.TrimPrefix(string(msg), "server: ")
			_, err := strconv.Atoi(portString)
			if err != nil {
				t.Fatalf("unexpected response from server: user id expected to be a number, got %s, err: %v", string(msg), err)
			}
		}
	}
}

func (c *TestClient) read() {
	for {
		_, msg, err := c.WS.ReadMessage()
		fmt.Println(string(msg))
		if err != nil {
			c.WS.Close()
			return
		}
		if len(msg) > 0 {
			c.Data <- msg
		}
	}
}
