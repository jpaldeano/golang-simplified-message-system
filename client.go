package main

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
	ws   *websocket.Conn
	data chan []byte
}

func initClient(address string) {
	var addr = flag.String("addr", address, "http service address")

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	client := &Client{ws: c}
	go client.read()

	for {
		reader := bufio.NewReader(os.Stdin)
		message, _ := reader.ReadString('\n')
		c.WriteMessage(1, []byte(strings.TrimRight(message, "\n")))
	}
}

func (c *Client) read() {
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			c.ws.Close()
			break
		}
		if len(msg) > 0 {
			fmt.Println("client read:", string(msg))
		}
	}
}
