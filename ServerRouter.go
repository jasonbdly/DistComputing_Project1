package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"math/rand"
	"time"
	p2p "./p2pmessage"
)

const (
	HOST        = ""
	PORT        = "5556"
	TYPE        = "tcp"
	SERVER_ROUTER = ""
)

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
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

var metricsChannel = make(chan Metric, 1)
var metricsSignalChannel = make(chan bool, 1)

//Tracks IP Addresses of all nodes
var nodeRegistry []string

var host string = ""
var serverPort string = ""
var sRouter_addr string

//Command line arguments: [HOST, PORT, OTHER_SERVER_ROUTER_ADDRESS]
func main() {
	if len(os.Args) == 4 {
		//Parse input arguments
		host = os.Args[1]
		serverPort = os.Args[2]
		sRouter_addr = os.Args[3]
	} else {
		fmt.Println("No CMD args detected. Using default settings.")
		host = HOST

		fmt.Print("Enter port to use (leave empty for default - WILL NOT WORK WITH MULTIPLE ROUTERS): ")
		reader := bufio.NewScanner(os.Stdin)

		//Block until the enter key is pressed, then read any new content into <text>
		reader.Scan()
		serverPort = reader.Text()
		if len(serverPort) == 0 {
			serverPort = PORT
		}

		//Print out a prompt to the client
		fmt.Print("Other Server Router Address (leave empty for default): ")

		//Block until the enter key is pressed, then read any new content into <text>
		reader.Scan()
		sRouter_addr = reader.Text()

		if len(sRouter_addr) == 0 {
			sRouter_addr = SERVER_ROUTER
		}
	}

	rand.Seed(time.Now().UnixNano() / int64(time.Millisecond))

	//Initialize the client-server routing registry
	nodeRegistry = []string{}

	p2p.ListenerPort = serverPort

	//Start concurrent session monitor
	//go MetricThread(metricsChannel, metricsSignalChannel)

	//Set up central listener
	listener, err := net.Listen(TYPE, host+":"+p2p.ListenerPort)
	checkErr(err, "Failed to create listener.")

	defer listener.Close()

	fmt.Println("[SERVERROUTER] LISTENING ON " + TYPE + "://" + host + ":" + p2p.ListenerPort)

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

		//Handle the connection in a separate thread
		go handleConnection(connection)
	}

	fmt.Println("[SERVERROUTER] SHUTTING DOWN")
}

func handleConnection(connection net.Conn) {
	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client

	defer connection.Close()

	if sRouter_addr == "" {
		fmt.Println("No other server routers found. Skipping this connection.")
		return
	}

	var msg p2p.Message
	dec := json.NewDecoder(connection)
	
	for {
		if err := dec.Decode(&msg); err != nil {
			p2p.TrackEOF()
			continue
		}
		
		msg.Conn = connection

		switch msg.Type {
			case p2p.IDENTIFY:
				fmt.Println("[SERVERROUTER] IDENTIFY FROM: " + msg.Src_IP)

				//Add sender to list of registered nodes
				nodeRegistry = append(nodeRegistry, msg.Src_IP)

				//Send acknowledgement packet back to node
				msg.Reply(p2p.ACKNOWLEDGE, "", msg.Src_IP)

				break
			case p2p.FIND_PEER:
				fmt.Println("[SERVERROUTER] FIND_PEER FROM: " + msg.Src_IP)

				// Send request to other server router to pick a node for the p2p connection
				findNodeResponse := p2p.Send(p2p.PICK_NODE, "", sRouter_addr)
				findNodeResponse.Conn.Close()

				// Send the IP of the picked node to the peer that originally requested a connection
				msg.Reply(p2p.FIND_PEER_RESPONSE, findNodeResponse.MSG, msg.Src_IP)

				break
			case p2p.PICK_NODE:
				fmt.Println("[SERVERROUTER] PICK_NODE FROM: " + msg.Src_IP)

				//pick random peer
				selected_peer_ip := nodeRegistry[rand.Intn(len(nodeRegistry))]
				msg.Reply(p2p.PICK_NODE_RESPONSE, selected_peer_ip, msg.Src_IP)

				break
		}
	}

	fmt.Println("[SERVERROUTER] CLOSING CONNECTION THREAD TO TYPE: " + msg.MSG)
}