package main

import (
	"fmt"
	"strings"
	"net"
	"bufio"
	"os"
)

const (
	HOST = "localhost"
	PORT = "5555"
	TYPE = "tcp"
)

func main() {
	connection, err := net.Dial(TYPE, HOST + ":" + PORT)

	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	defer connection.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Text to Send: ")
		
		text, _ := reader.ReadString('\n')

		text = strings.Trim(text, "\n")

		if len(text) > 0 {
			fmt.Fprintf(connection, text + "\n")

			message, _ := bufio.NewReader(connection).ReadString('\n')
			fmt.Print("Response: " + message)
		}
	}
}