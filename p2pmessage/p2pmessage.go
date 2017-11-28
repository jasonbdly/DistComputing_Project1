package p2pmessage

import (
	"encoding/json"
	"strconv"
	"fmt"
	"net"
	"os"
	"bufio"
	"runtime/debug"
	"math/rand"
	"time"
	metrics "../metricutil"
)

type MessageType int
const (
	ACKNOWLEDGE MessageType = iota
	IDENTIFY
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
	MAX_CONN_ATTEMPTS = 5
	MAX_PEER_IDLE_TIME = time.Second * 1
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

var lanAddress string
func GetLANAddress() string {
	if len(lanAddress) == 0 {
		addrs, err := net.InterfaceAddrs()

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					lanAddress = ipnet.IP.To4().String() + ":" + ListenerPort
					break
				}
			}
		}

		if len(lanAddress) == 0 {
			lanAddress = "localhost" + ":" + ListenerPort
		}
	}
	return lanAddress
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

func connectToTCP(address string) net.Conn {
	var conn net.Conn
	var err error

	fmt.Println("(" + GetLANAddress() + ") Attempting connnection to: (" + address + ")")

	//attemptConnectionStartTime := time.Now()

	connectionAttempts := 0
	for connectionAttempts < MAX_CONN_ATTEMPTS && conn == nil {
		err = nil
		conn, err = net.Dial("tcp", address)
		if err != nil {
			fmt.Println("(" + GetLANAddress() + ") Failed to connect to (" + address + ") Trying again")
			connectionAttempts++
		}
	}

	if conn == nil {
		fmt.Println("(" + GetLANAddress() + ") Failed to make a connection to (" + address + ")\n" + err.Error() + "\n" + string(debug.Stack()))
		os.Exit(1)
	}

	//metrics.AddVal("TCP_CONNECT", int64(time.Since(attemptConnectionStartTime)))

	fmt.Println("(" + GetLANAddress() + ") SUCCESSFUL connection to: " + address)

	return conn
}

//sends message to a peer
func Send(Type MessageType, MSG string, Rec_IP string) Message {
	fmt.Println("[SEND]: " + Type.String() + ", " + Rec_IP)

	msg := createMessage(Type, MSG, Rec_IP)

	connection := connectToTCP(Rec_IP)

	enc := json.NewEncoder(connection)
	enc.Encode(msg)

	// getting reply
	var reply_msg Message
	json.NewDecoder(connection).Decode(&reply_msg)

	reply_msg.Conn = connection

	return reply_msg
}

func Send_Scanner(Type MessageType, MSG *bufio.Scanner, Rec_IP string) <-chan Message {
	outputChannel := make(chan Message, 1)

	connection := connectToTCP(Rec_IP)

	dataEncoder := json.NewEncoder(connection)
	dataDecoder := json.NewDecoder(connection)

	go func() {
		sendDataStartTime := time.Now()
		lineCounter := 0
		for MSG.Scan() {
			nextLineData := MSG.Text()

			if len(nextLineData) > 0 {
				nextPacket := createMessage(Type, nextLineData, Rec_IP)
				dataEncoder.Encode(nextPacket)

				var replyPacket Message
				err := dataDecoder.Decode(&replyPacket)

				if err != nil {
					fmt.Println(err)
					break
				}

				sendDataElapsedTime := time.Since(sendDataStartTime)
				if sendDataElapsedTime == 0 {
					sendDataElapsedTime = 1
				}
				metrics.AddVal("P2P_SEND_DATA_NS", int64(sendDataElapsedTime))
				metrics.AddVal("P2P_TRANSMISSION_SIZE", int64(len(nextLineData)))
				metrics.AddVal("P2P_BYTES_PER_NS", int64(len(nextLineData)) / int64(sendDataElapsedTime))

				fmt.Println("SENT: " + nextLineData + "\n" + "RECEIVED: " + replyPacket.MSG + "\nTOOK: " + strconv.Itoa(int(sendDataElapsedTime)) + " ns")

				outputChannel <- replyPacket

				lineCounter++
			}
			sendDataStartTime = time.Now()
		}

		dataEncoder.Encode(createMessage(DISCONNECT, "", Rec_IP))

		close(outputChannel)

		connection.Close()
	}()

	return outputChannel
}

func (msg *Message) Reply(Type MessageType, MSG string, Rec_IP string) {
	fmt.Println("[REPLY]: " + msg.Type.String() + ", " + Rec_IP)

	if msg.Conn == nil {
		fmt.Println("CONNECTION NULL")
	}

	enc := json.NewEncoder(msg.Conn)
	enc.Encode(createMessage(Type, MSG, Rec_IP))
}