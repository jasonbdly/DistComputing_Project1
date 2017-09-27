package main

import (
	"fmt"
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
	connection, _ := net.Dial(TYPE, HOST + ":" + PORT)

	for {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Text to Send: ")
		
		text, _ := reader.ReadString('\n')

		fmt.Fprintf(connection, text + "\n")

		message, _ := bufio.NewReader(connection).ReadString('\n')
		fmt.Print("Response: " + message)
	}
}