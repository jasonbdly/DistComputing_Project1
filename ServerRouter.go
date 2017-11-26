package main

import (
	//"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	//"strings"
	//"time"
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

//Main thread logic
func main() {
	//Initialize the client-server routing registry
	//routingRegistry := []RoutingRegisterEntry{}

	nodeRegistry = []string{}

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

	var msg Message
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

			break
		case "QUERY":
			if len(msg.MSG) > 0 {
				fmt.Println("READ FROM PEER " + msg.Src_IP + ":" + msg.MSG)

				//Uppercase the message
				//message := strings.ToUpper(msg.MSG)

				//Write the uppercased message back to the remote connection
				//res_msg := createMessage("RESPONSE", getLANAddress(), message, msg.Src_IP)
				//res_msg.send(sRouter_addr)
			}
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

//message sent out to the server
type Message struct {
	Type   string //type of message ("IDENTIFY","RESPONSE","QUERY","ACK","DISCONNECT")
	Src_IP string //Ip address of my computer
	MSG    string //message
	Rec_IP string // IP address of message recipient
}

//creates a new message using the parameters passed in and returns it
func createMessage(Type string, Src_IP string, MSG string, Rec_IP string) (msg *Message) {
	msg = new(Message)
	msg.Type = Type
	msg.Src_IP = Src_IP
	msg.MSG = MSG
	msg.Rec_IP = Rec_IP
	return
}

//sends message to a peer
func (msg *Message) send(receiver string) {
	connection, err := net.Dial(TYPE, msg.Rec_IP)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(msg)
}