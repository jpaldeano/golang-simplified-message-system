package test

import (
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
	time.Sleep(time.Second * 1) // give some breathing room to receive the server response
	clientX = nil
}

func TestGetList(t *testing.T) {
	var clientX *TestClient

	address := "localhost:8081"
	go msgSystemHub.InitHub(address)
	clientX = newTestClient(address)
	_ = newTestClient(address) // create another client, otherwise only the client X will be connected and list will be empty

	go clientX.assertGetListResponseFromServer(t)
	go clientX.WS.WriteMessage(1, []byte("list"))
	time.Sleep(time.Second * 1) // give some breathing room to receive the server response
}

func TestRelay(t *testing.T) {
	testChannel := make(chan string)
	address := "localhost:8082"
	go msgSystemHub.InitHub(address)
	clientX := newTestClient(address)
	_ = newTestClient(address) // create another client, otherwise only the client X will be connected and list will be empty

	go clientX.assertRelayMessage(t, testChannel)
	go clientX.getUserIDsAndAttachRelay(testChannel)
	clientX.WS.WriteMessage(1, []byte("list"))
	time.Sleep(time.Second * 1) // give some breathing room to receive the server response

}

type TestClient struct {
	WS   *websocket.Conn
	Data chan []byte
}

func newTestClient(address string) *TestClient {
	u := url.URL{Scheme: "ws", Host: address, Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	client := &TestClient{WS: c, Data: make(chan []byte)}
	go client.read()

	return client

}

func (c *TestClient) getUserIDsAndAttachRelay(listUserIDChannel chan string) {
	for {
		select {
		case msg := <-listUserIDChannel:
			relayMsg := fmt.Sprintf("relay|users=%s,body=hello world", msg)
			c.WS.WriteMessage(1, []byte(relayMsg))
		}
	}
}

func (c *TestClient) assertRelayMessage(t *testing.T, listUserIDChannel chan string) {
	for {
		select {
		case msg := <-c.Data:
			if strings.HasPrefix(string(msg), "server: users list:") {
				portString := strings.TrimPrefix(string(msg), "server: users list: \n0) ")
				portString = strings.TrimSuffix(portString, "\n")
				_, err := strconv.Atoi(portString)
				if err != nil {
					t.Fatalf("unexpected response from server: user id expected to be a number, got %s, err: %v", string(msg), err)
				}
				listUserIDChannel <- portString
			} else {
				// the server does not respond to  user when he sends relay messages
				t.Fatal(msg)
			}
		}
	}
}

func (c *TestClient) assertGetIDResponseFromServer(t *testing.T) {
	for {
		select {
		case msg := <-c.Data:
			if !strings.HasPrefix(string(msg), "server: ") {
				t.Fatalf("unexpected response from server: expected to be prefixed by 'server: ', got %s", string(msg))
			}

		}

	}
}

func (c *TestClient) assertGetListResponseFromServer(t *testing.T) {
	for {
		select {
		case msg := <-c.Data:
			if !strings.HasPrefix(string(msg), "server: ") {
				t.Fatalf("unexpected response from server: expected to be prefixed by 'server: ', got %s", string(msg))
			}

			portString := strings.TrimPrefix(string(msg), "server: users list: \n0) ")
			portString = strings.TrimSuffix(portString, "\n")
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
