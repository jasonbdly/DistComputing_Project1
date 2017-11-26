package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"./p2pmessage"
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
	fmt.Println(getLANAddress())

	go server()      // Starting server thread

	time.Sleep(time.Second * 5)

	identifyMyself() // IDing self to server router to become available to peers

	time.Sleep(time.Second * 5)
	
	go client()      // Starting client thread
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
	serverRouterAddress := reader.Text()

	if len(serverRouterAddress) == 0 {
		serverRouterAddress = SERVER_ROUTER
	}
	sRouter_addr = serverRouterAddress

	id_msg := p2pmessage.CreateMessage("IDENTIFY", getLANAddress(), "", serverRouterAddress)
	id_msg.Send(serverRouterAddress)
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

				// query srouter for peer ip
				peer_ip_q_msg := p2pmessage.CreateMessage("QUERY", getLANAddress(), "", sRouter_addr) // creating query message to send

				peer_ip := peer_ip_q_msg.Send(sRouter_addr)
				/*peer_ip_q_msg.send(sRouter_addr)

				//receiving reply with peer ip
				listener, err := net.Listen(TYPE, HOST+":"+PORT)
				checkErr(err, "Error creating listener while requesting peer IP")

				//Queue the listener's Close behavior to be fired once this function scope closes
				defer listener.Close()

				//Block until a connection is received from a remote client
				connection, err := listener.Accept()
				checkErr(err, "Error accepting connection while requesting peer IP")

				var msg Message
				json.NewDecoder(connection).Decode(&msg)

				peer_ip := msg.Message //reply*/

				//send message to new ip
				q_msg := p2pmessage.CreateMessage("QUERY", getLANAddress(), text, peer_ip.MSG) // creating query message to send
				q_msg.Send(sRouter_addr)
				fmt.Println("Sent to " + peer_ip.MSG)
			}

		}
	} else {

		// query srouter for peer ip
		peer_ip_q_msg := p2pmessage.CreateMessage("QUERY", getLANAddress(), "", sRouter_addr) // creating query message to send
		peer_ip_res_msg := peer_ip_q_msg.Send(sRouter_addr)

		//receiving reply
		/*listener, err := net.Listen(TYPE, HOST+":"+PORT)
		checkErr(err, "Error creating listener")

		//Queue the listener's Close behavior to be fired once this function scope closes
		//defer listener.Close()

		listen_connection, err := listener.Accept()
		checkErr(err, "Error accepting connection")

		fmt.Println("NEW REQUEST FROM PEER ACCEPTED - HANDLING IN NEW THREAD")

		dec := json.NewDecoder(listen_connection)
		msg := new(p2pmessage.Message)

		peer_ip := msg.MSG //reply

		listener.Close()
		listen_connection.Close()*/

		connection, err := net.Dial(TYPE, peer_ip_res_msg.MSG)
		if err != nil {
			fmt.Println("Failed to create connection to the server. Is the server listening?")
			os.Exit(1)
		}

		//Defer closing the connection to the remote listener until this function's scope closes
		defer connection.Close()

		for _, element := range message_split { // for each element of string array
			element = strings.Trim(element, "\n") // trim end line char
			if len(element) > 0 {

				//send message to new ip
				q_msg := p2pmessage.CreateMessage("QUERY", getLANAddress(), text, peer_ip_res_msg.MSG) // creating query message to send
				q_msg.Send_Conn(connection)
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

func handleRequest(connection net.Conn) {

	if testing {
		log.Println("receive")
	}
	defer connection.Close()

	var msg p2pmessage.Message
	dec := json.NewDecoder(connection)
	for {
		if err := dec.Decode(&msg); err != nil {
			fmt.Println("ERROR: " + err.Error())
			return
		}
		switch msg.Type {
		case "IDENTIFY":
			// do nothing, shouldnt have gotten this
			break
		case "QUERY":
			if len(msg.MSG) > 0 {
				// message = strings.Trim(message, "\n") // should be trimmed already

				fmt.Println("READ FROM PEER " + msg.Src_IP + ":" + msg.MSG)

				//Uppercase the message
				message := strings.ToUpper(msg.MSG)

				//Write the uppercased message back to the remote connection
				res_msg := p2pmessage.CreateMessage("RESPONSE", getLANAddress(), message, msg.Src_IP)
				res_msg.Send(sRouter_addr)

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
