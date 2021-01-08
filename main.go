package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("wrong number of arguments - first argument should be 'hub' or 'client' and second argument should be the address:port we want connect to")
	} else {
		switch os.Args[1] {
		case "hub":
			InitHub(os.Args[2])
		case "client":
			initClient(os.Args[2])
		default:
			fmt.Println("first argument should be 'hub' or 'client'")
		}
	}
}
