package p2pmessage

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

type MessageType int
const (
	IDENTIFY MessageType = iota
	ACKNOWLEDGE
	DISCONNECT
	FIND_PEER
	FIND_PEER_RESPONSE
	PICK_NODE
	PICK_NODE_RESPONSE
	DATA
	DATA_RESPONSE
)

var ListenerPort string = ""

func GetLANAddress() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.To4().String() + ":" + ListenerPort
			}
		}
	}

	fmt.Println("Failed to retrieve LAN address")
	os.Exit(1)
	return "localhost" + ":" + ListenerPort
}

type Message struct {
	Type   MessageType //type of message ("IDENTIFY","RESPONSE","QUERY","IP_QUERY","ACK","DISCONNECT")
	Src_IP string //Ip address of my computer
	MSG    string //message
	Rec_IP string // IP address of message recipient
}

//creates a new message using the parameters passed in and returns it
func CreateMessage(Type MessageType, MSG string, Rec_IP string) (msg *Message) {
	msg = new(Message)
	msg.Type = Type
	msg.Src_IP = GetLANAddress()
	msg.MSG = MSG
	msg.Rec_IP = Rec_IP
	return msg
}

func (msg *Message) Send_Async() {
	//if testing {
		log.Println("send async(ip) to " + msg.Rec_IP)
		log.Println(msg.Src_IP)
	//}

	//connection, err := net.Dial(TYPE, serverRouterAddress)
	connection, err := net.Dial("tcp", msg.Rec_IP)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(msg)
}

//sends message to a peer
func (msg *Message) Send() Message {
	//if testing {
		log.Println("send(ip) to " + msg.Rec_IP)
		log.Println(msg.Src_IP)
	//}

	//connection, err := net.Dial(TYPE, serverRouterAddress)
	connection, err := net.Dial("tcp", msg.Rec_IP)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(msg)

	// getting reply
	var reply_msg Message
	json.NewDecoder(connection).Decode(&reply_msg)

	return reply_msg
}


//sends message to a peer
func (msg *Message) Send_Conn(conn net.Conn) {
	//if testing {
		log.Println("send(conn)")
	//}

	enc := json.NewEncoder(conn)
	enc.Encode(msg)
}