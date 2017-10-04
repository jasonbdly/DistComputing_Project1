package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
)

const (
	HOST = "localhost"
	PORT = "5555"
	TYPE = "tcp"
)

func getLANAddress() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.To4().String()
			}
		}
	}

	fmt.Println("Failed to retrieve LAN address")
	os.Exit(1)
}

func main() {
	//Set up a listener on the configured protocol, host, and port
	//listener, err := net.Listen(TYPE, HOST+":"+PORT)
	lanAddress := getLANAddress()
	listener, err := net.Listen(TYPE, lanAddress+":"+PORT)
	if err != nil {
		fmt.Println("Error creating listener: ", err.Error())
		os.Exit(1)
	}

	//Queue the listener's Close behavior to be fired once this function scope closes
	defer listener.Close()

	//fmt.Println("Listening on " + TYPE + "://" + HOST + ":" + PORT)
	fmt.Println("Listening on " + TYPE + "://" + lanAddress + ":" + PORT)

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

	fmt.Println("Server shutting down")
}

func handleRequest(connection net.Conn) {
	//Defer closing of the connection until this function's scope is closed
	defer connection.Close()

	clientConnectionDetails := connection.RemoteAddr()

	connectionIdStr := clientConnectionDetails.Network() + "://" + clientConnectionDetails.String()

	fmt.Println(connectionIdStr + " connected")

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

			if message == "EXIT" {
				break
			}

			fmt.Println("Received from [" + connectionIdStr + "]: " + message)

			//Uppercase the message
			transformedMessage := strings.ToUpper(message)

			//Write the uppercased message back to the remote connection
			connection.Write([]byte(transformedMessage + "\n"))

			fmt.Println("Sent to [" + connectionIdStr + "]: " + transformedMessage)
		} else {
			timesReceivedBlank++
		}

		//Cleanly handles closing the connection if a direct interrupt is used instead of sending "EXIT"
		if timesReceivedBlank == 5 {
			break
		}
	}

	fmt.Println(connectionIdStr + " disconnected")
}
