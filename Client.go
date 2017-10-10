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

const (
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)

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

var transmissionTimes = make([]time.Duration, 0) //List(slice) of transmission times

func main() {
	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewScanner(os.Stdin)

	//Print out a prompt to the client
	fmt.Print("Server Router Address (leave empty for default): ")

	//Block until the enter key is pressed, then read any new content into <text>
	reader.Scan()
	serverRouterAddress := reader.Text()

	if len(serverRouterAddress) == 0 {
		serverRouterAddress = SERVER_ROUTER
	}

	connection, err := net.Dial(TYPE, serverRouterAddress)
	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	serverConnectionDetails := connection.RemoteAddr()

	connectionIdStr := serverConnectionDetails.Network() + "://" + serverConnectionDetails.String()

	fmt.Println("Connected to: " + connectionIdStr)

	//Create a buffer to interface with the remote connection
	connReader := bufio.NewScanner(connection)

	connection.Write([]byte("CLIENT\n"))

	connReader.Scan()
	connReader.Text()

	fmt.Println("Connected to ServerRouter")

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
				fmt.Println("Sent to [" + connectionIdStr + "]: " + text)

				//Use the Fprintf to send the inputted text to the remote connection
				connection.Write([]byte(text + "\n"))

				// getting time message was sent to compare with time reply was received
				timeSent := time.Now()

				if text != "EXIT" {
					//Block until a newline character is received from the connection
					connReader.Scan()
					message := connReader.Text()

					fmt.Println("Received from [" + connectionIdStr + "]: " + message)

					if len(message) == 0 {
						break
					}
					fmt.Println(message)

					//Sdding transmission times to list (slice)
					transmissionTimes = append(transmissionTimes, time.Since(timeSent))
				} else {
					printTransmissionMetrics()
					break
				}
			}

		}
	} else {
		for _, element := range message_split {
			element = strings.Trim(element, "\n")

			if len(element) > 0 {
				fmt.Println("Sent to [" + connectionIdStr + "]: " + text)

				//Use the Fprintf to send the inputted text to the remote connection
				connection.Write([]byte(element + "\n"))

				// getting time message was sent to compare with time reply was received
				timeSent := time.Now()

				if element != "EXIT" {
					//Block until a newline character is received from the connection
					connReader.Scan()
					message := connReader.Text()

					fmt.Println("Received from [" + connectionIdStr + "]: " + message)

					if len(message) == 0 {
						break
					}

					//Sdding transmission times to list (slice)
					transmissionTimes = append(transmissionTimes, time.Since(timeSent))

					//Print out the response to the console
					//fmt.Println("Received from [" + connectionIdStr + "]: " + message)
				} else {
					break
				}
			}
		}

		printTransmissionMetrics()

		//Notify the server that we've started
		connection.Write([]byte("EXIT\n"))
		time.Sleep(10000 * time.Millisecond)
	}
}
