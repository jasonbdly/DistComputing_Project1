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
	//Set up a listener on the configured protocol, host, and port
	listener, err := net.Listen(TYPE, HOST + ":" + PORT)
	if err != nil {
		fmt.Println("Error creating listener: ", err.Error())
		os.Exit(1)
	}

	//Queue the listener's Close behavior to be fired once this function scope closes
	defer listener.Close()

	fmt.Println("Created listener on " + HOST + ":" + PORT)
	
	//Essentially a while(true) loop
	for {
		//Block until a connection is received from a remote client
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		//Run the "handleRequest" function for this connection in a separate thread
		go handleRequest(connection)
	}
}

func handleRequest(connection net.Conn) {
	//Defer closing of the connection until this function's scope is closed
	defer connection.Close()

	//Set up buffered reader for the new connection
	connectionReader := bufio.NewReader(connection)

	timesReceivedBlank := 0

	//Continue listening for messages from the remote client until "EXIT" is received
	for {
		//Read the next line from the connection's input buffer
		message, _ := connectionReader.ReadString('\n')

		message = strings.Trim(message, "\n")

		if len(message) != 0 {
			timesReceivedBlank = 0

			if strings.Trim(message, " \n") == "EXIT" {
				break
			}

			fmt.Println("Received Message: " + message)

			//Uppercase the message
			transformedMessage := strings.ToUpper(message)

			//Write the uppercased message back to the remote connection
			connection.Write([]byte(transformedMessage + "\n"))
		} else {
			timesReceivedBlank++
		}

		//Cleanly handles closing the connection if a direct interrupt is used instead of sending "EXIT"
		if timesReceivedBlank == 5 {
			break
		}
	}
}