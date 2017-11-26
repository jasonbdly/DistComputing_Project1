package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

var HOST = ""
var sRouter_addr = ""
var transmissionTimes = make([]time.Duration, 0) //List(slice) of transmission times

const (
	PORT          = "5557"
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)

//message sent out to the server
type Message struct {
	Type   string //type of message ("IDENTIFY","RESPONSE","QUERY","ACK","DISCONNECT")
	src_IP string //Ip address of my computer
	MSG    string //message
	rec_IP string // IP address of message recipient
}

func main() {
	identifyMyself() // IDing self to server router to become available to peers
	go server()      // Starting server thread
	go client()      // Starting client thread
}

//creates a new message using the parameters passed in and returns it
func createMessage(Type string, src_IP string, MSG string, rec_IP string) (msg *Message) {
	msg = new(Message)
	msg.Type = Type
	msg.src_IP = src_IP
	msg.MSG = MSG
	msg.rec_IP = rec_IP
	return
}

//sends message to a peer
func (msg *Message) send(receiver string) {
	if testing {
		log.Println("sendPrivate")
	}

	connection, err := net.Dial(TYPE, serverRouterAddress)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(msg)
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

func introduceMyself() {
	//Print out a prompt to the client
	fmt.Print("Server Router Address (leave empty for default): ")

	//Block until the enter key is pressed, then read any new content into <text>
	reader.Scan()
	serverRouterAddress := reader.Text()

	if len(serverRouterAddress) == 0 {
		serverRouterAddress = SERVER_ROUTER
	}
	sRouter_addr = serverRouterAddress

	id_msg = createMessage("IDENTIFY", getLANAddress(), "", serverRouterAddress)
	id_msg.send(serverRouterAddress)
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

	if useTerminal {
		for {
			fmt.Print("Text to Send: ")
			//Block until the enter key is pressed, then read any new content into <text>
			reader.Scan()
			text = reader.Text()

			if len(text) > 0 {
				q_msg = createMessage("QUERY", getLANAddress(), make([]string, text), sRouter_addr) // creating query message to send
				q_msg.send(sRouter_addr)                                                            // sending
			}

		}
	} else {
		for _, element := range message_split { // for each element of string array
			element = strings.Trim(element, "\n") // trim end line char
			if len(element) > 0 {
				q_msg = createMessage("QUERY", getLANAddress(), make([]string, element), sRouter_addr) // creating query messsage to send
				q_msg.send(sRouter_addr)                                                               // sending

			}
		}

		//printTransmissionMetrics()
	}
}

func server() {
	reader := bufio.NewScanner(os.Stdin)

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

func handleRequest(connection net.Conn) {

	if testing {
		log.Println("receive")
	}
	defer conn.Close()
	dec := json.NewDecoder(conn)
	msg := new(Message)
	for {
		if err := dec.Decode(msg); err != nil {
			return
		}
		switch msg.Kind {
		case "IDENTIFY":
			// do nothing, shouldnt have gotten this
			break
		case "QUERY":
			if len(message) > 0 {
				// message = strings.Trim(message, "\n") // should be trimmed already

				fmt.Println("READ FROM PEER " + msg.IP + ":" + message)

				//Uppercase the message
				message := strings.ToUpper(message)

				//Write the uppercased message back to the remote connection
				res_msg = createMessage("RESPONSE", getLANAddress(), message, msg.src_IP)
				res_msg.send(sRouter_addr)

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

}

func checkErr(err error, message string) {
	if err != nil {
		fmt.Println(message, err.Error())
		os.Exit(1)
	}
}
