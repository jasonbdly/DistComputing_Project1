package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"math/rand"
	//"strings"
	"time"
	p2p "./p2pmessage"
)

const (
	HOST        = ""
	PORT        = "5556"
	TYPE        = "tcp"
	SERVER_PORT = "5557"
	SERVER_ROUTER = ""
)

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}

type RoutingRegisterEntry struct {
	ServerAddr string
	NumClients int
}

type IncomingConnection struct {
	Type       string
	Connection *net.Conn
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

//Tracks IP Addresses of all nodes
var nodeRegistry []string
var sRouter_addr string

//Main thread logic
func main() {
	p2p.ListenerPort = PORT

	rand.Seed(time.Now().UnixNano() / int64(time.Millisecond))

	//Initialize the client-server routing registry
	//routingRegistry := []RoutingRegisterEntry{}

	nodeRegistry = []string{}

	//Print out a prompt to the client
	fmt.Print("Other Server Router Address (leave empty for default): ")

	reader := bufio.NewScanner(os.Stdin)

	//Block until the enter key is pressed, then read any new content into <text>
	reader.Scan()
	serverRouterAddress := reader.Text()

	if len(serverRouterAddress) == 0 {
		serverRouterAddress = SERVER_ROUTER
	}
	sRouter_addr = serverRouterAddress

	//Start concurrent session monitor
	//go MetricThread(metricsChannel, metricsSignalChannel)

	//Set up central listener
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Failed to create listener.")

	defer listener.Close()

	fmt.Println("[SERVERROUTER] LISTENING ON " + TYPE + "://" + HOST + ":" + PORT)
	fmt.Println("[SERVERROUTER] LAN ADDRESS: " + p2p.GetLANAddress())

	/*go func() {
		for {
			metricsSignalChannel <- true
			time.Sleep(5 * time.Second)
		}
	}()*/

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

var numErrors int = 0
func handleConnection(connection net.Conn) {
	fmt.Println("[SERVERROUTER] WAITING FOR CONNECTION TYPE MESSAGE")

	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client

	defer connection.Close()

	var msg p2p.Message
	dec := json.NewDecoder(connection)
	//msg := new(Message)

	for {
		if err := dec.Decode(&msg); err != nil {
			fmt.Println("[SERVERROUTER] ERROR - Failed to parse packet - Skipping")
			numErrors++

			if numErrors >= 5 {
				fmt.Println("[SERVERROUTER] TOO MANY ERRORS. CLOSING SERVER ROUTER")
				break
			}

			continue
		}

		numErrors = 0

		switch msg.Type {
			case p2p.IDENTIFY:
				fmt.Println("[SERVERROUTER] RECEIVED IDENTIFY FROM NODE: " + msg.Src_IP)

				//Add sender to list of registered nodes
				nodeRegistry = append(nodeRegistry, msg.Src_IP)

				//Send acknowledgement packet back to node
				acknowledgementMessage := p2p.CreateMessage(p2p.ACKNOWLEDGE, "", msg.Src_IP)
				acknowledgementMessage.Send_Async()

				break
			case p2p.FIND_PEER:
				// Send request to other server router to pick a node for the p2p connection
				msg := p2p.CreateMessage(p2p.PICK_NODE, "", sRouter_addr)
				findNodeResponse := msg.Send()

				// Send the IP of the picked node to the peer that originally requested a connection
				peerResponseMessage := p2p.CreateMessage(p2p.FIND_PEER_RESPONSE, findNodeResponse.MSG, msg.Src_IP)
				peerResponseMessage.Send_Async()
				break
			case p2p.PICK_NODE:
				//pick random peer
				selected_peer_ip := nodeRegistry[rand.Intn(len(nodeRegistry))]
				msg := p2p.CreateMessage(p2p.PICK_NODE_RESPONSE, selected_peer_ip, msg.Src_IP)
				msg.Send_Async()

				break
		}
	}

	fmt.Println("[SERVERROUTER] CLOSING CONNECTION THREAD TO TYPE: " + msg.MSG)
}