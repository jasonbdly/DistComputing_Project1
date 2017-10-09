package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
	"os"
	"strings"
)

const (
	HOST = "localhost"
	PORT = "5556"
	TYPE = "tcp"
)

func printTransmissionMetrics(){
	var total_time time.Duration = 0 * time.Second
	fmt.Println("Transmission Rate Metrics: ");
	for index, element := range transmissionTimes {
			total_time = total_time + element
        	//fmt.Print("Transmission #[%v]: [%v]", index, element)
        	fmt.Print("Transmission #")
        	fmt.Print(index)
        	fmt.Print(": ")
        	fmt.Println(element)
    }
    fmt.Print("Average transmission time (in seconds): ")
    fmt.Println(int(total_time) / 1000 /len(transmissionTimes))
    return

}

var transmissionTimes = make([]time.Duration,0) //List(slice) of transmission times
func main() {
	
	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewReader(os.Stdin)

	//Print out a prompt to the client
	fmt.Print("Server Router Address: ")
	
	//Block until the enter key is pressed, then read any new content into <text>
	newHost,err:= reader.ReadString('\n')
	newHost = strings.Trim(newHost, "\n")
	//Attempt to connect to a listener on HOST:PORT via the TYPE protocol
	connection, err := net.Dial(TYPE, newHost+":"+PORT)

	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	fmt.Fprintf(connection, "CLIENT\n")

	serverConnectionDetails := connection.RemoteAddr()

	connectionIdStr := serverConnectionDetails.Network() + "://" + serverConnectionDetails.String()

	fmt.Println("Connected to: " + connectionIdStr)


	//Create a buffer to interface with the remote connection
	connReader := bufio.NewReader(connection)

	//Essentially a while(true) loop
	for {
		//Print out a prompt to the client
		fmt.Print("Text to Send: ")

		//Block until the enter key is pressed, then read any new content into <text>
		text, _ := reader.ReadString('\n')

		//Trim the "newline" character from the read text
		text = strings.Trim(text, "\n")

		//Only handle the text is the text isn't empty
		if len(text) > 0 {
			fmt.Println("Sent to [" + connectionIdStr + "]: " + text)

			//Use the Fprintf to send the inputted text to the remote connection
			fmt.Fprintf(connection, text+"\n")
			// getting time message was sent to compare with time reply was received
			timeSent := time.Now();

			if text != "EXIT" {
				//Block until a newline character is received from the connection
				message, _ := connReader.ReadString('\n')
				//Sdding transmission times to list (slice)
				transmissionTimes = append(transmissionTimes, time.Since(timeSent))
				//Print out the response to the console
				fmt.Println("Received from [" + connectionIdStr + "]: " + message)
			} else {
				printTransmissionMetrics()
				break
			}
		}
	}
}
