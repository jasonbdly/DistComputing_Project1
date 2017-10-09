package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	HOST = "localhost"
	PORT = "5555"
	TYPE = "tcp"
	ROUTER_PORT = "5556"
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
	return ""
}

func main() {
	//Set up a listener on the configured protocol, host, and port
	//listener, err := net.Listen(TYPE, HOST+":"+PORT)
	//fmt.Println("0")

///*
	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewReader(os.Stdin)

	//Print out a prompt to the client
	fmt.Print("Server Router Address (leave empty for default): ")
	
	//Block until the enter key is pressed, then read any new content into <text>
	input,err:= reader.ReadString('\n')
	input = strings.Trim(input, "\n")
	
	var newHost string = ""
	if(len(input) == 0){
		newHost = HOST
	}else{
		newHost = input
	}

	fmt.Println("newHost = " +  newHost)


	//connection, err := net.Dial("tcp", "localhost:5556")
	connection, err := net.Dial(TYPE, newHost+":"+PORT)
//*/
	//connection, err := net.Dial(TYPE, HOST+":"+PORT)
	fmt.Fprintf(connection, "SERVER\n") // updating routing regristry
	connection.Close()

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

	//fmt.Println("server handle request...")
	//Defer closing of the connection until this function's scope is closed
	defer connection.Close()

	// getting remote network address
	clientConnectionDetails := connection.RemoteAddr()

	//	connectionIdStr = network address name://network address
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
	//fmt.Println("...server handle request")
}

