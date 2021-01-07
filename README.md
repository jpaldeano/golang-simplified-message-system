# golang-simplified-message-system

A message delivery system written using websockets.

## Hub
This implementation communicates via websockets. When the Hub starts by creating a http server - it upgrades the request so basically it layers on top of TCP and only uses http on the handshake phase.
The Hub relays on the websocket remote address of the incoming messages, new clients connect socket bind for client using port:0 and so server will give a random open port to the client (https://unix.stackexchange.com/questions/55913/whats-the-easiest-way-to-find-an-unused-local-port).

The server it keeps the connected clients on a map where the key is the port and the value the client. 

> go run *.go hub {address}:{port}

## Client
> go run *.go client {hubAddress:port}

### Types of messages accepted
The message delivery system includes the following possible message types from requesting client (clientX):

- **id** - (clientX->hub->clientX) the client can send an identity message which the hub will answer with the user id of the requesting client.
- **list** - (clientX->hub->clientX) the client can send a list message which the hub will answer with the list of all connected client user ids. 

- **relay|users=clientY;clientZ,body=hello chaps!** - (clientX-> [server->clientY & server->clientZ]) The client can send a relay message which body is relayed to receivers marked in the message. 