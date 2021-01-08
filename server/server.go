package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	client "github.com/jpaldi/golang-simplified-message-system/client"
)

const (
	maxBodySize            = 1024000
	maxReceiversPerMessage = 255
)

// HubMessage provides an helper to parse message and client details to the channel
type HubMessage struct {
	contents []byte
	client   *client.Client
}

// Hub represents the server node. Which is able to receive and send messages to clients via websocket
type Hub struct {
	upgrader        websocket.Upgrader     // websocket to upgrade
	messagesChannel chan *HubMessage       // messageChannel is used to read messages sent from clients
	connect         chan *client.Client    // connect is used to notify when a client connects
	disconnect      chan *client.Client    // disconnect is used to notify when a client disconnects
	clients         map[int]*client.Client // clients keeps connected clients
}

// InitHub starts an http server on the provided address and upgrades the connection to websockets
func InitHub(addr string) {
	fmt.Println("Starting hub on", addr)
	hub := Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		messagesChannel: make(chan *HubMessage),
		connect:         make(chan *client.Client),
		disconnect:      make(chan *client.Client),
		clients:         make(map[int]*client.Client),
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

	client := &client.Client{WS: conn, Data: make(chan []byte)}
	hub.connect <- client

	go hub.read(client)
	go hub.write(client)
}

func (hub *Hub) handle() {
	for {
		select {
		case connection := <-hub.connect:
			add := connection.WS.RemoteAddr().String()
			port, err := getPortFromAddress(add)
			if err != nil {
				fmt.Printf("connection error: %v", err)
				hub.disconnect <- connection
				return
			}
			hub.clients[*port] = connection // port is used in the map to identify client
			fmt.Printf("A new client connected with the hub from %s\n", connection.WS.RemoteAddr().String())
		case disconnect := <-hub.disconnect:
			close(disconnect.Data)
			fmt.Printf("Client %s closed connection with the hub\n", disconnect.WS.RemoteAddr().String())

		case message := <-hub.messagesChannel:
			hub.handleMessage(message)
		}
	}
}

func (hub *Hub) handleMessage(hubM *HubMessage) {
	add := hubM.client.WS.RemoteAddr().String()
	// hubM.client.data <- []byte(fmt.Sprintf("I received your message %s", add))

	port, err := getPortFromAddress(add)
	if err != nil {
		fmt.Printf("connection error: %v", err)
		hub.disconnect <- hubM.client
		return
	}

	msgStr := string(hubM.contents)
	fmt.Printf("from %s: %s\n", add, msgStr)

	if msgStr == "id" {
		hubM.client.Data <- []byte(fmt.Sprint(*port))
		return
	}

	if msgStr == "list" {
		usersList := hub.getAllUsersExcept(*port)
		hubM.client.Data <- clientsToBytes(usersList)
		return
	}

	if strings.HasPrefix(msgStr, "relay") {
		// The client can send a list message which the hub will answer with the list of all connected client user_id:s (excluding the requesting client).
		hub.parseRelayString(hubM)
		return
	}

	hubM.client.Data <- []byte("command not recognized")
}
func (hub *Hub) parseRelayString(message *HubMessage) {
	// relay|users=u1;u2,body=con
	relay := strings.TrimPrefix(string(message.contents), "relay|")

	relayArgs := strings.Split(relay, ",")
	if len(relayArgs) != 2 {
		message.client.Data <- []byte("relay message should contain users and body fields")
		return
	}

	if !strings.HasPrefix(relayArgs[0], "users=") {
		message.client.Data <- []byte("relay message should contain users field")
		return
	}

	if !strings.HasPrefix(relayArgs[1], "body=") {
		message.client.Data <- []byte("relay message should contain a body field")
		return
	}
	users := strings.TrimPrefix(relayArgs[0], "users=")
	body := strings.TrimPrefix(relayArgs[1], "body=")

	destList := strings.Split(users, ";")
	if len(destList) == 0 {
		message.client.Data <- []byte("unexpected message format")
		return
	}

	if len(destList) > maxReceiversPerMessage {
		message.client.Data <- []byte("max receivers per message exceeded")
		return
	}

	if len(body) > maxBodySize {
		message.client.Data <- []byte("message body can't exceed 1024kb")
		return
	}

	senderID, _ := getPortFromAddress(message.client.WS.RemoteAddr().String())
	for _, u := range destList {
		userID, _ := strconv.Atoi(u)
		destClient, found := hub.clients[userID]
		if !found {
			// if user in the provided list can't be found, return to the client the error
			message.client.Data <- []byte(fmt.Sprintf("userid not found: %s", u))
		} else {
			// if user in the provided list is active, send the message and attach the user that sent it
			userName := []byte(fmt.Sprintf("%d-> ", *senderID))
			destClient.Data <- append(userName, body...)
		}
	}
}

func clientsToBytes(clients []*client.Client) []byte {
	value := []byte("users list: \n")
	for i, c := range clients {
		id, _ := getPortFromAddress(c.WS.RemoteAddr().String())
		bValue := append([]byte(fmt.Sprint(i)+") "), []byte(fmt.Sprint(*id))...)
		bValue = append(bValue, []byte("\n")...)
		value = append(value, bValue...)
	}
	return value
}

func (hub *Hub) getAllUsersExcept(user int) []*client.Client {
	clients := make([]*client.Client, 0, len(hub.clients))
	for k, v := range hub.clients {
		// exclude itself from list
		if k != user {
			clients = append(clients, v)
		}
	}
	return clients
}

func getPortFromAddress(a string) (*int, error) {
	portStr := strings.Split(a, ":")
	if len(portStr) != 2 {
		return nil, fmt.Errorf("error reading the address: %s", a)
	}
	port, err := strconv.Atoi(portStr[1])
	if err != nil {
		return nil, fmt.Errorf("error converting port: %s", portStr[1])
	}
	return &port, nil
}

func (hub *Hub) read(client *client.Client) {
	for {
		_, msg, err := client.WS.ReadMessage()
		if err != nil {
			hub.disconnect <- client
			client.WS.Close()
			break
		}
		if len(msg) > 0 {
			hub.messagesChannel <- &HubMessage{contents: msg, client: client}
		}

	}
}

func (hub *Hub) write(client *client.Client) {
	for {
		select {
		case message, ok := <-client.Data:
			if !ok {
				return
			}
			client.WS.WriteMessage(1, append([]byte("server: "), message...))
		}
	}
}
