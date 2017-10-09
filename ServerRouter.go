package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	HOST        = ""
	PORT        = "5556"
	TYPE        = "tcp"
	SERVER_PORT = "5557"
)

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}

type RoutingRegisterEntry struct {
	ServerConnection  net.Conn
	ClientConnections []net.Conn

	ServerAddr string
	NumClients int
}

type IncomingConnection struct {
	Type       string
	Connection *net.Conn
}

func connectionAssigner(incomingChannel chan IncomingConnection, assignedChannel chan<- string, routingRegistry *[]RoutingRegisterEntry) {
	queuedClients := []IncomingConnection{}

	for {
		//Block until a new client connection request is received
		newConnection := <-incomingChannel

		fmt.Println("[ASSIGNER] NEW CONNECTION: " + newConnection.Type)

		if newConnection.Type == "SERVER" {
			remoteConnectionAddress := (*newConnection.Connection).RemoteAddr().String()
			remoteAddressParts := strings.Split(remoteConnectionAddress, ":")
			remoteAddressParts = remoteAddressParts[:len(remoteAddressParts)-1]

			remoteAddressIP := strings.Join(remoteAddressParts, ":")

			//Server
			*routingRegistry = append(*routingRegistry, RoutingRegisterEntry{ServerAddr: remoteAddressIP + ":" + SERVER_PORT, NumClients: 0})

			fmt.Println("[ASSIGNER] SEND CONNECTED SIGNAL TO SERVER")
			(*newConnection.Connection).Write([]byte("CONNECTED\n"))
			defer (*newConnection.Connection).Close()
		} else {
			//We'll need to wait until a server has connected before assigning any clients
			if len(*routingRegistry) > 0 {
				//Client
				var bestServerEntry *RoutingRegisterEntry
				for _, serverEntry := range *routingRegistry {
					if bestServerEntry == nil || serverEntry.NumClients < bestServerEntry.NumClients {
						bestServerEntry = &serverEntry
					}
				}

				if len(queuedClients) > 0 {
					//Remove that entry from the queued clients list
					queuedClients = append(queuedClients[:1], queuedClients[2:]...)
				}

				bestServerEntry.NumClients++

				fmt.Println("[ASSIGNER] SERVER ASSIGNED TO CLIENT")
				assignedChannel <- bestServerEntry.ServerAddr
			} else {
				queuedClients = append(queuedClients, newConnection)
			}
		}
	}
}

//Set up channel for synchronized communication with the
//connection mapper channel. A buffer size of 1 is used to
//ensure connection mapping requests are handled in order.
//Golang channels block sends when the buffer is full, and block
//receives when the buffer is empty. Channels are the preferred method
//of cross-thread communication in place of manual synchronization with
//more standard data structures.
var newConnectionChannel = make(chan IncomingConnection, 1)
var assignedChannel = make(chan string, 1)

//Main thread logic
func main() {
	//Initialize the client-server routing registry
	routingRegistry := []RoutingRegisterEntry{}

	//Start the server assigner thread
	go connectionAssigner(newConnectionChannel, assignedChannel, &routingRegistry)

	//Set up central listener
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Failed to create listener.")

	defer listener.Close()

	//fmt.Println("Server Router listening on " + TYPE + "://" + HOST + ":" + PORT)
	fmt.Println("[SERVERROUTER] LISTENING ON " + TYPE + "://" + HOST + ":" + PORT)

	for {
		//Wait for a connection
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection.")

		fmt.Println("[SERVERROUTER] NEW CONNECTION ACCEPTED - HANDLING IN NEW THREAD")

		//Handle the connection in a separate thread
		go handleConnection(connection)
	}

	//fmt.Println("Server Router stopping")
	fmt.Println("[SERVERROUTER] SHUTTING DOWN")
}

func handleConnection(connection net.Conn) {
	fmt.Println("[SERVERROUTER] WAITING FOR CONNECTION TYPE MESSAGE")

	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client
	connectionReader := bufio.NewScanner(connection)

	connectionReader.Scan()
	firstMessage := connectionReader.Text()

	firstMessage = strings.Trim(firstMessage, "\n")

	fmt.Println("[SERVERROUTER] RECEIVED CONNECTION TYPE MESSAGE: " + firstMessage)

	newConnectionChannel <- IncomingConnection{Type: firstMessage, Connection: &connection}

	//Only need to facilitate connection if the connector is a "client" type
	//Otherwise, it must be a server, so we just keep track of it to set up routing
	//between it and other clients.
	if firstMessage == "CLIENT" {
		defer connection.Close()

		fmt.Println("[SERVERROUTER] CLIENT TYPE DETECTED - STARTING HANDLER THREAD")

		assignedServerAddr := <-assignedChannel

		fmt.Println("[SERVERROUTER] CONNECTING CLIENT TO: " + assignedServerAddr)

		serverConnection, err := net.Dial(TYPE, assignedServerAddr)
		checkErr(err, "Failed to open proxy connection to server")

		defer serverConnection.Close()

		serverReader := bufio.NewScanner(serverConnection)

		//Signal to the client that it has been successfully routed to a server
		fmt.Println("[SERVERROUTER] NOTIFYING CLIENT OF CONNECTION TO SERVER")
		connection.Write([]byte("CONNECTED\n"))

		//Start the main proxy loop to facilitate communication
		for {
			connectionReader.Scan()
			checkErr(connectionReader.Err(), "Failed to read from client")

			message := connectionReader.Text()
			message = strings.Trim(message, "\n")

			fmt.Println("[SERVERROUTER] READ FROM CLIENT: " + message)

			serverConnection.Write([]byte(message + "\n"))

			fmt.Println("[SERVERROUTER] WROTE TO SERVER: " + message)

			//Break this connection if the client sent a disconnect signal
			if message == "EXIT" {
				break
			}

			serverReader.Scan()
			serverMessage := serverReader.Text()

			fmt.Println("[SERVERROUTER] READ FROM SERVER: " + serverMessage)

			//Send the server's response back to the client
			connection.Write([]byte(serverMessage + "\n"))

			fmt.Println("[SERVERROUTER] WROTE TO CLIENT: " + serverMessage)
		}
	}

	fmt.Println("[SERVERROUTER] CLOSING CONNECTION THREAD TO TYPE: " + firstMessage)
}
