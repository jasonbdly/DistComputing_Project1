package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"log"
	"net"
	"os"
	"strings"
	"time"
	p2p "./p2pmessage"
)

var HOST = ""
var sRouter_addr = ""
var transmissionTimes = make([]time.Duration, 0) //List(slice) of transmission times

var testing bool = true

const (
	PORT          = "5557"
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)

func main() {
	p2p.ListenerPort = PORT

	go server()      // Starting server thread

	time.Sleep(time.Second * 5)

	identifyMyself() // IDing self to server router to become available to peers

	time.Sleep(time.Second * 5)
	
	go client()      // Starting client thread
}

func printTransmissionMetrics() {
	var total_time time.Duration = 0 * time.Second
	fmt.Println("Transmission Rate Metrics: ")
	for index, element := range transmissionTimes {
		total_time = total_time + element
		//fmt.Print("Transmission #[%v]: [%v]", index, element)
		fmt.Print("Transmission #")
		fmt.Print(index)
		fmt.Print(": ")
		fmt.Println(element)
	}
	fmt.Print("Average transmission time (in seconds): ")
	fmt.Println(int(total_time) / 1000 / len(transmissionTimes))
	return
}

func identifyMyself() {
	//Print out a prompt to the client
	fmt.Print("Server Router Address (leave empty for default): ")

	reader := bufio.NewScanner(os.Stdin)

	//Block until the enter key is pressed, then read any new content into <text>
	reader.Scan()
	sRouter_addr := reader.Text()

	if len(sRouter_addr) == 0 {
		sRouter_addr = SERVER_ROUTER
	}

	id_msg := p2p.CreateMessage(p2p.IDENTIFY, "", sRouter_addr)
	id_msg.Send()
	// wait for ack?
}

func client() {
	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewScanner(os.Stdin)

	var text string = ""
	var useTerminal bool = true
	fmt.Print("Enter path to file you would like to transmit (leave empty to enter custom text): ")
	reader.Scan()
	path := reader.Text()

	message_split := []string{}
	if len(path) != 0 {

		text_array, _ := ioutil.ReadFile(path)    // getting file
		text = string(text_array)                 // setting to string
		message_split = strings.Split(text, "\n") // splitting string at new line chars
		useTerminal = false
	}

	// Instruct parent server router to find a peer for this node, via communicating with the other server router
	findPeerMessage := p2p.CreateMessage(p2p.FIND_PEER, "", sRouter_addr)
	foundPeerResponseMessage := findPeerMessage.Send()

	//Retrieve the picked node from the packet
	peerNode := foundPeerResponseMessage	.MSG

	if useTerminal {
		for {
			fmt.Print("Text to Send: ")
			//Block until the enter key is pressed, then read any new content into <text>
			reader.Scan()
			text = reader.Text()

			if len(text) > 0 {
				if text == "EXIT" {
					break
				}

				//Send data from user to peer node, block until the response is received, and print the response
				dataMessage := p2p.CreateMessage(p2p.DATA, text, peerNode)
				dataResponseMessage := dataMessage.Send()
				fmt.Println("RESPONSE: " + dataResponseMessage.MSG)
			}
		}
	} else {
		/*connection, err := net.Dial(TYPE, peer_ip_res_msg.MSG)
		if err != nil {
			fmt.Println("Failed to create connection to the server. Is the server listening?")
			os.Exit(1)
		}

		//Defer closing the connection to the remote listener until this function's scope closes
		defer connection.Close()*/

		for _, element := range message_split { // for each element of string array
			element = strings.Trim(element, "\n") // trim end line char
			if len(element) > 0 {

				//Send data from user to peer node, block until the response is received, and print the response
				dataMessage := p2p.CreateMessage(p2p.DATA, element, peerNode)
				dataResponseMessage := dataMessage.Send()
				fmt.Println("RESPONSE: " + dataResponseMessage.MSG)

				//send message to new ip
				//q_msg := p2p.CreateMessage("QUERY", text, peer_ip_res_msg.MSG) // creating query message to send
				//q_msg.Send_Conn(connection)
			}
		}

		//printTransmissionMetrics()
	}
}

func server() {
	//reader := bufio.NewScanner(os.Stdin)

	//Set up a listener on the configured protocol, host, and port
	listener, err := net.Listen(TYPE, HOST+":"+PORT)
	checkErr(err, "Error creating listener")

	//Queue the listener's Close behavior to be fired once this function scope closes
	defer listener.Close()

	fmt.Println("STARTED LISTENING ON " + TYPE + "://" + HOST + ":" + PORT)

	//Essentially a while(true) loop
	for {
		//Block until a connection is received from a remote client
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection")

		fmt.Println("NEW REQUEST FROM PEER ACCEPTED - HANDLING IN NEW THREAD")

		//Run the "handleRequest" function for this connection in a separate thread
		go handleRequest(connection)
	}

	fmt.Println("SHUTTING DOWN SERVER THREAD - PEERS WON'T BE ABLE TO CONNECT")
}

var numErrors int = 0
func handleRequest(connection net.Conn) {
	defer connection.Close()

	var msg p2p.Message
	dec := json.NewDecoder(connection)

	for {
		if err := dec.Decode(&msg); err != nil {
			fmt.Println("ERROR: " + err.Error())
			numErrors++

			if numErrors >= 5 {
				fmt.Println("[P2P NODE] TOO MANY ERRORS. CLOSING PROGRAM")
				break;
			}

			continue
		}

		numErrors = 0

		switch msg.Type {
			case p2p.DATA:
				if len(msg.MSG) > 0 {
					fmt.Println("READ FROM PEER " + msg.Src_IP + ":" + msg.MSG)

					//Uppercase the message
					message := strings.ToUpper(msg.MSG)

					//Write the uppercased message back to the remote connection
					res_msg := p2p.CreateMessage(p2p.DATA_RESPONSE, message, msg.Src_IP)
					res_msg.Send_Async()
				}
				break
			case p2p.DISCONNECT:
				
				break
		}
	}
}

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}
