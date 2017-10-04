package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	HOST = "localhost"
	PORT = "5556"
	TYPE = "tcp"
)

func checkErr(err error.Error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}

type RoutingRegisterEntry struct {
	ServerConnection  net.Conn
	ClientConnections []net.Conn
}

type IncomingConnection struct {
	Type string
	Connection net.Conn
}

func connectionAssigner(incomingChannel chan IncomingConnection, assignedChannel chan <- net.Conn, routingRegistry []RoutingRegisterEntry) {
	queuedClients := []net.Conn{}

	for {
		//Block until a new client connection request is received
		newConnection := <-incomingChannel
		
		if newConnection.Type == "SERVER" {
			//Server
			routingRegistry = append(routingRegistry, &RoutingRegisterEntry{
				ServerConnection: newConnection,
				ClientConnections: []net.Conn{}
			})
		} else {
			//We'll need to wait until a server has connected before assigning any clients
			if len(routingRegistry) > 0 {
				//Client
				var bestServerEntry RoutingRegisterEntry
				for _, serverEntry := range routingRegister {
					if bestServerEntry == nil || len(serverEntry.ClientConnections) < len(bestServerEntry.ClientConnections) {
						bestServerEntry = serverEntry
					}
				}
				
				var clientConnection net.Conn

				if len(queuedClients) > 0 {
					//Pull the first entry from the queued clients list
					clientConnection = queuedClients[0]

					//Remove that entry from the queued clients list
					queuedClients = append(queuedClients[:1], queuedClients[2:]...)
				} else {
					clientConnection = newConnection.Connection
				}

				bestServerEntry.ClientConnections = append(bestServerEntry.ClientConnections, clientConnection)

				assignedChannel <- bestServerEntry.ServerConnection
			} else {
				queuedClients = append(queuedClients, newConnection.Connection)
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
var assignedChannel = make(chan net.Conn, 1)

//Main thread logic
func main() {
	//Initialize the client-server routing registry
	routingRegistry := []RoutingRegisterEntry{}

	//Start the server assigner thread
	go connectionAssigner(newConnectionChannel, assignedChannel, routingRegistry)

	//Set up central listener process
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Failed to create listener.")
	defer listener.Close()
	fmt.Println("Server Router listening on " + TYPE + "://" + HOST + ":" + PORT)

	for {
		//Wait for a connection
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection.")

		//Handle the connection in a separate thread
		go handleConnection(connection)
	}

	fmt.Println("Server Router stopping")
}

func handleConnection(connection net.Conn) {
	defer connection.Close()

	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client
	connectionReader := bufio.NewReader(connection)

	firstMessage, err := connectionReader.ReadString('\n')
	checkErr(err, "Failed to read from remote connection")

	firstMessage = strings.Trim(firstMessage, "\n")
	newConnectionChannel <- &IncomingConnection{
		Type: firstMessage,
		Connection: connection
	}

	//Only need to facilitate connection if the connector is a "client" type
	//Otherwise, it must be a server, so we just keep track of it to set up routing
	//between it and other clients.
	if firstMessage == "CLIENT" {
		assignedServer <- assignedChannel

		serverReader := bufio.NewReader(assignedServer)

		//Signal to the client that it has been successfully routed to a server
		connection.Write([]byte("CONNECTED\n"))

		//Start the main proxy loop to facilitate communication
		for {
			message, err := connectionReader.ReadString('\n')
			checkErr(err, "Failed to read from client")

			message = strings.Trim(message, "\n")

			assignedServer.Write([]byte(message + "\n"))

			//Break this connection if the client sent a disconnect signal
			if message == "EXIT" {
				break;
			}

			serverMessage, err := serverReader.ReadString('\n')
			checkErr(err, "Failed to read from server")

			//Send the server's response back to the client
			connection.Write([]byte(serverMessage))
		}
	}
}
