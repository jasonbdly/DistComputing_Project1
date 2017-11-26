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
	"./p2pmessage"
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

func getLANAddress() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.To4().String() + ":" + PORT
			}
		}
	}

	fmt.Println("Failed to retrieve LAN address")
	os.Exit(1)
	return "localhost" + ":" + PORT
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
	fmt.Println("[SERVERROUTER] LAN ADDRESS: " + getLANAddress())

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


func handleConnection(connection net.Conn) {
	fmt.Println("[SERVERROUTER] WAITING FOR CONNECTION TYPE MESSAGE")

	//Wait for the initialization packet, which specifies whether the remote machine
	//is a server or a client

	var msg p2pmessage.Message
	dec := json.NewDecoder(connection)
	//msg := new(Message)

	for {
		if err := dec.Decode(&msg); err != nil {
			return
		}

		msgDetails, _ := json.Marshal(msg)
		fmt.Println("MESSAGE: " + string(msgDetails))

		switch msg.Type {
		case "IDENTIFY":
			fmt.Println("[SERVERROUTER] RECEIVED IDENTIFY FROM NODE: " + msg.Src_IP)

			//Add sender to list of registered nodes
			nodeRegistry = append(nodeRegistry, msg.Src_IP)

			acknowledgementMessage := p2pmessage.CreateMessage("ACK", getLANAddress(), "", msg.Src_IP)
			acknowledgementMessage.Send_Async("")

			break
		case "QUERY":
			if len(msg.MSG) > 0 {
				fmt.Println("READ FROM PEER " + msg.Src_IP + ":" + msg.MSG)

				// send hard coded srouter ip IP_QUERY
				msg := p2pmessage.CreateMessage("IP_QUERY", getLANAddress(), "", sRouter_addr)
				msg.Send_Async(sRouter_addr)

				//receiving reply with peer ip
				/*listener, err := net.Listen(TYPE, HOST+":"+PORT)
				checkErr(err, "Error creating listener while requesting peer IP")

				//Queue the listener's Close behavior to be fired once this function scope closes
				defer listener.Close()

				//Block until a connection is received from a remote client
				connection, err := listener.Accept()
				checkErr(err, "Error accepting connection while requesting peer IP")

				dec := json.NewDecoder(connection)
				msg := new(Message)

				peer_ip := msg.Message //reply

*/
				//Uppercase the message
				//message := strings.ToUpper(msg.MSG)

				//Write the uppercased message back to the remote connection
				//res_msg := createMessage("RESPONSE", getLANAddress(), message, msg.Src_IP)
				//res_msg.send(sRouter_addr)
			}
			break
		case "IP_QUERY":
			fmt.Println(msg.MSG) // printing capitalized text

			//pick random peer
			selected_peer_ip := nodeRegistry[rand.Intn(len(nodeRegistry))]
			msg := p2pmessage.CreateMessage("RESPONSE", getLANAddress(), selected_peer_ip, sRouter_addr)
			msg.Send_Async(sRouter_addr)
			 
			break
		case "RESPONSE":
			fmt.Println(msg.MSG) // printing capitalized text
			break
		case "ACK":
			// do nothing
			break
		}
	}

	fmt.Println("[SERVERROUTER] CLOSING CONNECTION THREAD TO TYPE: " + msg.MSG)
}