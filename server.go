package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// HubMessage provides an helper to parse message and client details to the channel
type HubMessage struct {
	contents []byte
	client   *Client
}

// Hub represents the server node. Which is able to receive and send messages to clients via websocket
type Hub struct {
	messagesChannel chan *HubMessage
	upgrader        websocket.Upgrader
	connect         chan *Client
}

func initHub(addr string) {
	fmt.Println("Starting hub on", addr)
	hub := Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connect:         make(chan *Client),
		messagesChannel: make(chan *HubMessage),
	}
	go hub.handle()

	r := mux.NewRouter()
	r.HandleFunc("/ws", hub.serveWS)
	log.Fatal(http.ListenAndServe(addr, r))
}

func (hub *Hub) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := hub.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	client := &Client{ws: conn, data: make(chan []byte)}
	hub.connect <- client

	go hub.read(client)
	go hub.write(client)
}

func (hub *Hub) handle() {
	for {
		select {
		case connection := <-hub.connect:
			fmt.Printf("A new client connected with the hub from %s\n", connection.ws.RemoteAddr().String())
		case message := <-hub.messagesChannel:
			fmt.Printf("from %s: %s\n", message.client.ws.RemoteAddr().String(), string(message.contents))
			// reply to client
			message.client.data <- []byte("server: I received your message")
		}
	}
}

func (hub *Hub) read(client *Client) {
	for {
		_, msg, err := client.ws.ReadMessage()
		if err != nil {
			client.ws.Close()
			break
		}
		if len(msg) > 0 {
			hub.messagesChannel <- &HubMessage{contents: msg, client: client}
		}

	}
}

func (hub *Hub) write(client *Client) {
	for {
		select {
		case message, ok := <-client.data:
			if !ok {
				return
			}
			client.ws.WriteMessage(1, message)
		}
	}
}
