package main

import (
	"fmt"
	"net"
	"os"
	"bufio"
	"strings"
)

const (
	HOST = "localhost"
	PORT = "5555"
	TYPE = "tcp"
)

func main() {
	listener, err := net.Listen(TYPE, HOST + ":" + PORT)
	if err != nil {
		fmt.Println("Error creating listener: ", err.Error())
		os.Exit(1)
	}

	//Queue the listener's Close behavior to be fired once this function returns
	defer listener.Close()

	fmt.Println("Created listener on " + HOST + ":" + PORT)
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("Started request handler thread")
		go handleRequest(connection)
		fmt.Println("After started request handler thread")
	}
}

func handleRequest(connection net.Conn) {
	for {
		message, _ := bufio.NewReader(connection).ReadString('\n')

		if message.Trim() == "EXIT" {
			break
		}

		fmt.Println("Received Message: " + message);

		transformedMessage := strings.ToUpper(message)

		connection.Write([]byte(transformedMessage + "\n"))
	}

	connection.Close()
}