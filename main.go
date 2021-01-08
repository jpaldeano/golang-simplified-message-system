package main

import (
	"fmt"
	"os"

	client "github.com/jpaldi/golang-simplified-message-system/client"
	hub "github.com/jpaldi/golang-simplified-message-system/server"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("wrong number of arguments - first argument should be 'hub' or 'client' and second argument should be the address:port we want connect to")
	} else {
		switch os.Args[1] {
		case "hub":
			hub.InitHub(os.Args[2])
		case "client":
			client.InitClient(os.Args[2])
		default:
			fmt.Println("first argument should be 'hub' or 'client'")
		}
	}
}
