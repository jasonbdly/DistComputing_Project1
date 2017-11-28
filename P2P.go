package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"strconv"
	"time"
	p2p "./p2pmessage"
	metrics "./metricutil"
)

var host string = ""
var sRouter_addr string = ""
var testFile string = ""
var transmissionTimes = make([]time.Duration, 0) //List(slice) of transmission times
var initialWaitTime int = 2000

const (
	HOST = ""
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)


//Command line arguments: [HOST, PORT, SERVER_ROUTER_ADDRESS, TEST_FILE, WaitTime]
func main() {
	if len(os.Args) == 6 {
		//Parse input arguments
		host = os.Args[1]
		p2p.ListenerPort = os.Args[2]
		sRouter_addr = os.Args[3]
		testFile = os.Args[4]
		initialWaitTime, _ = strconv.Atoi(os.Args[5])
		fmt.Println("[P2P]: " + host + ", " + os.Args[2] + ", " + sRouter_addr + ", " + testFile + ", " + os.Args[5])
	} else {
		fmt.Println("No CMD args detected. Using default settings.")
		host = HOST

		p2p.ListenerPort = p2p.FindOpenPort(host, 49152, 65535)
		
		//Print out a prompt to the client
		fmt.Print("Server Router Address (leave empty for default): ")

		reader := bufio.NewScanner(os.Stdin)

		//Block until the enter key is pressed, then read any new content into <text>
		reader.Scan()
		sRouter_addr = reader.Text()

		if len(sRouter_addr) == 0 {
			sRouter_addr = SERVER_ROUTER
		}
	}

	metrics.Start()

	fmt.Println("[P2P] (" + host + ":" + p2p.ListenerPort + ") starting with router: " + sRouter_addr)

	go server()      // Starting server thread

	time.Sleep(time.Duration(initialWaitTime) * time.Millisecond)

	identifyStartTime := time.Now()
	// IDing self to server router to become available to peers
	identifyResponse := p2p.Send(p2p.IDENTIFY, "", sRouter_addr)

	metrics.SetVal("P2P_IDENTIFY_NS", int64(time.Since(identifyStartTime)))

	identifyResponse.Conn.Close()
	
	client()      // Starting client thread

	//Wait for server to be idle before stopping
	for {
		if time.Since(lastPeerInteractionTime) > p2p.MAX_PEER_IDLE_TIME {
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	//Write metric results to file
	metrics.Stop("./metrics/P2P/node_" + host + "_" + p2p.ListenerPort + ".json")
}

func client() {
	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewScanner(os.Stdin)

	var text string = ""
	var useTerminal bool = true

	if testFile == "" {
		fmt.Print("Enter path to file you would like to transmit (leave empty to enter custom text): ")
		reader.Scan()
		testFile = reader.Text()
	}

	if len(testFile) != 0 {
		text_array, _ := ioutil.ReadFile(testFile)    // getting file
		text = string(text_array)                 // setting to string
		useTerminal = false
	}

	var peerNode string

	findPeerStartTime := time.Now()

	for len(peerNode) == 0 {
		time.Sleep(100 * time.Millisecond)

		// Instruct parent server router to find a peer for this node, via communicating with the other server router
		foundPeerResponseMessage := p2p.Send(p2p.FIND_PEER, "", sRouter_addr)
		foundPeerResponseMessage.Conn.Close()

		//Retrieve the picked node from the packet
		peerNode = foundPeerResponseMessage.MSG
	}

	metrics.SetVal("P2P_FIND_PEER_NS", int64(time.Since(findPeerStartTime)))

	fmt.Println("[P2P] (" + host + ":" + p2p.ListenerPort + ") Got Peer: " + peerNode)

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

				sendDataStartTime := time.Now()
				//Send data from user to peer node, block until the response is received, and print the response
				dataResponseMessage := p2p.Send(p2p.DATA, text, peerNode)

				sendDataElapsedTime := time.Since(sendDataStartTime)

				metrics.AddVal("P2P_SEND_DATA_NS", int64(sendDataElapsedTime))
				metrics.AddVal("P2P_TRANSMISSION_SIZE", int64(len(text)))
				metrics.AddVal("P2P_BYTES_PER_NS", int64(len(text)) / int64(sendDataElapsedTime))

				dataResponseMessage.Conn.Close()
				fmt.Println("RESPONSE: " + dataResponseMessage.MSG)
			}
		}
	} else {
		fileTransmissionStartTime := time.Now()

		//NOTE: Look in p2pmessage.go - Send_Scanner for "per-line" metric code

		fileReader := bufio.NewScanner(strings.NewReader(text))
		responseChannel := p2p.Send_Scanner(p2p.DATA, fileReader, peerNode)

		for {
			_, hasMore := <-responseChannel
			if !hasMore {
				break
			}
		}

		fileTransmissionElapsedTime := time.Since(fileTransmissionStartTime)
		metrics.AddVal("P2P_SEND_FILE_NS", int64(fileTransmissionElapsedTime))
		metrics.AddVal("P2P_BYTES_PER_NS", int64(len(text)) / int64(fileTransmissionElapsedTime))
	}
}

var lastPeerInteractionTime time.Time = time.Now()
func server() {
	//Set up a listener on the configured protocol, host, and port
	listener, err := net.Listen(TYPE, host+":"+p2p.ListenerPort)
	checkErr(err, "Error creating listener")

	//Queue the listener's Close behavior to be fired once this function scope closes
	defer listener.Close()

	fmt.Println("[P2P] LISTENING ON " + TYPE + "://" + host + ":" + p2p.ListenerPort)

	//Essentially a while(true) loop
	for {
		//Block until a connection is received from a remote client
		connection, err := listener.Accept()
		checkErr(err, "Error accepting connection")

		lastPeerInteractionTime = time.Now()

		//Run the "handleRequest" function for this connection in a separate thread
		go handleRequest(connection)
	}

	fmt.Println("[P2P] SHUTTING DOWN")
}

func handleRequest(connection net.Conn) {
	defer connection.Close()

	var msg p2p.Message
	dec := json.NewDecoder(connection)

	LISTENER:
	for {
		if err := dec.Decode(&msg); err != nil {
			p2p.TrackEOF()
			continue
		}

		lastPeerInteractionTime = time.Now()

		msg.Conn = connection

		switch msg.Type {
			case p2p.DATA:
				//Write the uppercased message back to the remote connection
				msg.Reply(p2p.DATA_RESPONSE, strings.ToUpper(msg.MSG), msg.Src_IP)

				metrics.AddVal("P2P_REPLY_DATA_NS", int64(time.Since(lastPeerInteractionTime)))
				break
			case p2p.DISCONNECT:
				break LISTENER
		}
	}

	lastPeerInteractionTime = time.Now()
}

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}
