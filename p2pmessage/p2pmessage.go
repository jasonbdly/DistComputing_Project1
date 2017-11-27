package p2pmessage

import (
	"encoding/json"
	"strconv"
	"fmt"
	"log"
	"net"
	"os"
	"math/rand"
	"time"
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

func (t MessageType) String() string {
	switch (t) {
	case IDENTIFY:
		return "IDENTIFY"
	case ACKNOWLEDGE:
		return "ACKNOWLEDGE"
	case DISCONNECT:
		return "DISCONNECT"
	case FIND_PEER:
		return "FIND_PEER"
	case FIND_PEER_RESPONSE:
		return "FIND_PEER_RESPONSE"
	case PICK_NODE:
		return "PICK_NODE"
	case PICK_NODE_RESPONSE:
		return "PICK_NODE_RESPONSE"
	case DATA:
		return "DATA"
	case DATA_RESPONSE:
		return "DATA_RESPONSE"
	}
	return ""
}

const (
	ACCEPTABLE_EOFS = 5
	EOF_WAIT_TIME = time.Second * 2
)

var numEOFs int = 0
func TrackEOF() {
	numEOFs++
	if numEOFs >= ACCEPTABLE_EOFS {
		time.Sleep(EOF_WAIT_TIME)
		numEOFs = 0
	}
}

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

func FindOpenPort(host string, startRange int, endRange int) string {
	//Try to set up a listener on the picked port. If this fails, then another process
	//must be listening to that port
	port := strconv.Itoa(rand.Intn(endRange - startRange) + startRange);
	
	listener, err := net.Listen("tcp", host+":"+port)
	for err != nil {
		port = strconv.Itoa(rand.Intn(endRange - startRange) + startRange);
		listener, err = net.Listen("tcp", host+":"+port)
	}

	listener.Close()

	return port
}

type Message struct {
	Type   MessageType //type of message ("IDENTIFY","RESPONSE","QUERY","IP_QUERY","ACK","DISCONNECT")
	Src_IP string //Ip address of my computer
	MSG    string //message
	Rec_IP string // IP address of message recipient
	Conn net.Conn
}

//creates a new message using the parameters passed in and returns it
func createMessage(Type MessageType, MSG string, Rec_IP string) (msg *Message) {
	msg = new(Message)
	msg.Type = Type
	msg.Src_IP = GetLANAddress()
	msg.MSG = MSG
	msg.Rec_IP = Rec_IP
	return msg
}

func Send_Async(Type MessageType, MSG string, Rec_IP string) {
	log.Println("[SEND ASYNC]: " + Type.String() + ", " + Rec_IP)

	connection, err := net.Dial("tcp", Rec_IP)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(createMessage(Type, MSG, Rec_IP))
}

//sends message to a peer
func Send(Type MessageType, MSG string, Rec_IP string) Message {
	log.Println("[SEND]: " + Type.String() + ", " + Rec_IP)

	connection, err := net.Dial("tcp", Rec_IP)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	//defer connection.Close()

	enc := json.NewEncoder(connection)
	enc.Encode(createMessage(Type, MSG, Rec_IP))

	// getting reply
	var reply_msg Message
	json.NewDecoder(connection).Decode(&reply_msg)

	reply_msg.Conn = connection

	return reply_msg
}

func (msg *Message) Reply(Type MessageType, MSG string, Rec_IP string) {
	log.Println("[REPLY]: " + msg.Type.String() + ", " + Rec_IP)

	if msg.Conn == nil {
		fmt.Println("CONNECTION NULL")
	}

	enc := json.NewEncoder(msg.Conn)
	enc.Encode(createMessage(Type, MSG, Rec_IP))
}