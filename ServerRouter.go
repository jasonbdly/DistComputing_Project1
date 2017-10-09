package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	HOST = "localhost"
	PORT = "5556"
	TYPE = "tcp"
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
}

type IncomingConnection struct {
	Type       string
	Connection net.Conn
}

func connectionAssigner(incomingChannel chan IncomingConnection, assignedChannel chan<- net.Conn, routingRegistry *[]RoutingRegisterEntry) {
	queuedClients := []net.Conn{}

	for {
		//Block until a new client connection request is received
		newConnection := <-incomingChannel

		if newConnection.Type == "SERVER" {
			//fmt.Println("server connection incoming");
			//Server
			*routingRegistry = append(*routingRegistry, RoutingRegisterEntry{ServerConnection: newConnection.Connection, ClientConnections: []net.Conn{}})
			//Creating slice for storing number of connections for this server
			if(len(totalServerConnections) > 0){
				totalServerConnections[len(totalServerConnections) - 1] = make([]int,0)
			}else{
				totalServerConnections[0] = make([]int,0)
			}
		} else {
			//fmt.Print("type: ")
			//fmt.Print(newConnection.Type)
			//fmt.Println("-")
			//fmt.Println("client connection incoming");
			//We'll need to wait until a server has connected before assigning any clients
			if len(*routingRegistry) > 0 {
				//fmt.Println("serving clients")
				//Client
				var bestServerEntry *RoutingRegisterEntry
				for _, serverEntry := range *routingRegistry {
					if bestServerEntry == nil || len(serverEntry.ClientConnections) < len(bestServerEntry.ClientConnections) {
						bestServerEntry = &serverEntry
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
				//fmt.Println("queuing clients")
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
	go connectionAssigner(newConnectionChannel, assignedChannel, &routingRegistry)

	//Start concurrent session monitor
	go averageConnectionsMonitor(routingRegistry)

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
	printConnectionAveragesMetrics()
}

var totalConnections = make([]int, 0) //Slice (list) of number of total connections for all servers every 250 ms
var totalServerConnections = make([][]int,1) //Total number of client connections on a per server basis
func averageConnectionsMonitor(entries []RoutingRegisterEntry) {
//func averageConnectionsMonitor(*[]RoutingRegisterEntry entries){
	//entries := make([]RoutingRegisterEntry,0)
	//Creating ticker (reoccuring timer)
	ticker := time.NewTicker(time.Millisecond * 250)
	go func(){
        for _ = range ticker.C {
			total := 0
        	//Finding total number of connections
            for index, element := range entries{
            	//Computing total
				total = total + len(element.ClientConnections)
				//Adding current total to server connections slice
				totalServerConnections[index] = append(totalServerConnections[index], len(element.ClientConnections))
			}
			//Adding total to slice
			totalConnections = append(totalConnections, total)
        }
    }()

    print_ticker := time.NewTicker(time.Millisecond * 5000)
	go func(){
        for _ = range print_ticker.C {
        	printConnectionAveragesMetrics()
        }
    }()
    
	
}

func printConnectionAveragesMetrics(){
	
	fmt.Println("\nConcurrent Connection Metrics:")
	// Printing total
	total := 0
	for _, element := range totalConnections{
		total = total + element
    }
    fmt.Print("\tTotal average connections : ");
    fmt.Println(total/len(totalConnections) )

    // Printing server totals
    for index, element := range totalServerConnections{
    	totalServer := 0
    	for _, element := range element{
    		totalServer = totalServer + element
    	}
	    fmt.Print("\tAverage connections for server #" )
	    fmt.Print(index)
	    fmt.Print(" : ")
	    fmt.Print(totalServer/len(totalServerConnections))
	    
	    //fmt.Println("\tAverage connections for server #[%d]: [%d]", index, totalServer/len(totalServerConnections))
    }
    return
}

func printAverageMessageMetrics(){
	fmt.Println("Message Size Metrics:")
	// Printing total
	total := 0
	for index, element := range messageSizeList{
		//fmt.Print("Message #[%d] size: [%d]", index, element)
		fmt.Print("Message #")
		fmt.Print(index)
		fmt.Print(" size:")
		fmt.Println(element)
		total = total + element
    }

    fmt.Print("Average message size: ")
    fmt.Println( total/ len(messageSizeList))
    return
}

var messageSizeList = make([]int,0) //Slize(list) of message sizes
func handleConnection(connection net.Conn) {
	//fmt.Println("handle connection...")
	defer connection.Close()

	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client
	connectionReader := bufio.NewReader(connection)

	firstMessage, err := connectionReader.ReadString('\n')
	checkErr(err, "Failed to read from remote connection")

	firstMessage = strings.Trim(firstMessage, "\n")
	newConnectionChannel <- IncomingConnection{Type: firstMessage, Connection: connection}

	//Only need to facilitate connection if the connector is a "client" type
	//Otherwise, it must be a server, so we just keep track of it to set up routing
	//between it and other clients.
	if firstMessage == "CLIENT" {
		assignedServer := <-assignedChannel

		serverReader := bufio.NewReader(assignedServer)

		//Signal to the client that it has been successfully routed to a server
		connection.Write([]byte("CONNECTED\n"))
		
		//Start the main proxy loop to facilitate communication
		for {
			message, err := connectionReader.ReadString('\n')
			checkErr(err, "Failed to read from client")

			message = strings.Trim(message, "\n")
			//fmt.Println("message:" + message);

			//Adding size of current message to list of sizes
			messageSizeList = append(messageSizeList, len(message))

			assignedServer.Write([]byte(message + "\n"))
			//Break this connection if the client sent a disconnect signal
			if message == "EXIT" {
				break
			}

			serverMessage, err := serverReader.ReadString('\n')
			checkErr(err, "Failed to read from server")

			//Send the server's response back to the client
			connection.Write([]byte(serverMessage))
		}
	}
	//fmt.Println("...handle connection")
}
