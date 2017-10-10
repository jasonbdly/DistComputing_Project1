package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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

type RoutingRegisterEntry struct {
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

			metricsChannel <- Metric{Type: "SERVER_CONNECTIONS", Value: 1, OP: "TOTAL"}
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

				metricsChannel <- Metric{Type: "CLIENT_CONNECTIONS[" + bestServerEntry.ServerAddr + "]", Value: 1, OP: "TOTAL"}

				fmt.Println("[ASSIGNER] SERVER ASSIGNED TO CLIENT")
				assignedChannel <- bestServerEntry.ServerAddr
			} else {
				queuedClients = append(queuedClients, newConnection)
			}
		}
	}
}

type Metric struct {
	Type  string
	Value int64
	OP    string
	Units string
}

func MetricThread(metricChannel <-chan Metric, calcMetricsSignal <-chan bool) {
	metricsByType := map[string][]Metric{}

	for {
		select {
		case metricData := <-metricChannel:
			if metricsByType[metricData.Type] == nil {
				metricsByType[metricData.Type] = []Metric{}
			}
			metricsByType[metricData.Type] = append(metricsByType[metricData.Type], metricData)
		case calcSignal := <-calcMetricsSignal:
			if calcSignal {
				finalMetrics := []string{}

				//CALC METRICS
				for metricType := range metricsByType {
					metricDataForType := metricsByType[metricType]

					operation := metricDataForType[0].OP
					metricType := metricDataForType[0].Type
					metricUnits := metricDataForType[0].Units

					var totalValue int64 = 0
					for _, metricData := range metricDataForType {
						totalValue += metricData.Value
					}

					if operation == "TOTAL" {
						finalMetrics = append(finalMetrics, operation+" - "+metricType+" = "+strconv.FormatInt(totalValue, 10)+" "+metricUnits)
					} else if operation == "AVG" {
						finalMetrics = append(finalMetrics, operation+" - "+metricType+" = "+strconv.FormatInt(totalValue/int64(len(metricDataForType)), 10)+" "+metricUnits)
					}
				}

				fmt.Println("\n===== METRICS =====")
				for _, finalMetric := range finalMetrics {
					fmt.Println(finalMetric)
				}
			}
		default:
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

var metricsChannel = make(chan Metric, 1)
var metricsSignalChannel = make(chan bool, 1)

//Main thread logic
func main() {
	//Initialize the client-server routing registry
	routingRegistry := []RoutingRegisterEntry{}

	//Start the server assigner thread
	go connectionAssigner(newConnectionChannel, assignedChannel, &routingRegistry)

	//Start concurrent session monitor
	go MetricThread(metricsChannel, metricsSignalChannel)

	//Set up central listener
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Failed to create listener.")

	defer listener.Close()

	fmt.Println("[SERVERROUTER] LISTENING ON " + TYPE + "://" + HOST + ":" + PORT)
	fmt.Println("[SERVERROUTER] LAN ADDRESS: " + getLANAddress())

	go func() {
		for {
			metricsSignalChannel <- true
			time.Sleep(5 * time.Second)
		}
	}()

	for {
		//Wait for a connection
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection.")

		fmt.Println("[SERVERROUTER] NEW CONNECTION ACCEPTED - HANDLING IN NEW THREAD")

		//Handle the connection in a separate thread
		go handleConnection(connection)
	}

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

			//Adding size of current message to list of sizes
			metricsChannel <- Metric{Type: "MESSAGE_SIZE", Value: int64(len(message)), OP: "AVG"}

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
