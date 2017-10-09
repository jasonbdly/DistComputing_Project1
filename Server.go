package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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
	return "localhost"
}

//var HOST = getLANAddress()
var HOST = ""

const (
	PORT          = "5557"
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}

func main() {
	reader := bufio.NewScanner(os.Stdin)

	//Print out a prompt to the client
	fmt.Print("Server Router Address (leave empty for default): ")

	//Block until the enter key is pressed, then read any new content into <text>
	reader.Scan()
	serverRouterAddress := reader.Text()

	if len(serverRouterAddress) == 0 {
		serverRouterAddress = SERVER_ROUTER
	}
	fmt.Println("[SERVER] CONNECTING TO ROUTER")

	//Attempt to connect to the ServerRouter
	routerConnection, err := net.Dial(TYPE, serverRouterAddress)
	checkErr(err, "Failed to connect to ServerRouter")

	//Notify the server that we've started
	routerConnection.Write([]byte("SERVER\n"))

	fmt.Println("[SERVER] SENT DISCOVER MESSAGE TO ROUTER")

	routerReader := bufio.NewScanner(routerConnection)

	routerReader.Scan()
	routerReader.Text()

	fmt.Println("[SERVER] RECEIVED DISCOVERY ACCEPTANCE FROM ROUTER - CLOSING CONNECTION")

	routerConnection.Close()

	//Set up a listener on the configured protocol, host, and port
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Error creating listener")

	//Queue the listener's Close behavior to be fired once this function scope closes
	defer listener.Close()

	fmt.Println("[SERVER] STARTED LISTENING ON " + TYPE + "://" + HOST + ":" + PORT)

	//Essentially a while(true) loop
	for {
		//Block until a connection is received from a remote client
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection")

		fmt.Println("[SERVER] NEW CONNECTION ACCEPTED - HANDLING IN NEW THREAD")

		//Run the "handleRequest" function for this connection in a separate thread
		go handleRequest(connection)
	}

	fmt.Println("[SERVER] SHUTTING DOWN")
}

func handleRequest(connection net.Conn) {

	//fmt.Println("server handle request...")
	//Defer closing of the connection until this function's scope is closed
	defer connection.Close()

	// getting remote network address
	clientConnectionDetails := connection.RemoteAddr()

	//	connectionIdStr = network address name://network address
	connectionIdStr := clientConnectionDetails.Network() + "://" + clientConnectionDetails.String()

	fmt.Println("[SERVER] CONNECTION THREAD STARTED FOR " + connectionIdStr)

	//Set up buffered reader for the new connection
	connectionReader := bufio.NewScanner(connection)

	//Continue listening for messages from the remote client until "EXIT" is received
	for {
		fmt.Println("[SERVER] WAITING FOR MESSAGE")

		//Read the next line from the connection's input buffer
		connectionReader.Scan()
		message := connectionReader.Text()

		if len(message) > 0 {
			message = strings.Trim(message, "\n")

			fmt.Println("[SERVER] READ FROM CLIENT: " + message)

			if message == "EXIT" {
				break
			}

			//Uppercase the message
			message := strings.ToUpper(message)

			//Write the uppercased message back to the remote connection
			connection.Write([]byte(message + "\n"))

			fmt.Println("[SERVER] WROTE TO CLIENT: " + message)
		}
	}

	fmt.Println("[SERVER] CLOSING CONNECTION THREAD")
}
