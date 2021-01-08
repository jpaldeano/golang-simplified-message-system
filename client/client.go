package client

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// Client provides a client object to connect to server via websocket
type Client struct {
	WS   *websocket.Conn
	Data chan []byte
}

// InitClient provides a client that connects via websockets with the server hosted on the given address and path /ws
func InitClient(address string) {
	var addr = flag.String("addr", address, "http service address")

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	client := &Client{WS: c}
	go client.read()

	for {
		reader := bufio.NewReader(os.Stdin)
		message, _ := reader.ReadString('\n')
		c.WriteMessage(1, []byte(strings.TrimRight(message, "\n")))
	}
}

func (c *Client) read() {
	for {
		_, msg, err := c.WS.ReadMessage()
		if err != nil {
			c.WS.Close()
			return
		}
		if len(msg) > 0 {
			fmt.Println(string(msg))
		}
	}
}
