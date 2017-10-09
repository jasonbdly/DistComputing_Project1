package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	TYPE          = "tcp"
	SERVER_ROUTER = "localhost:5556"
)

func main() {
	//Attempt to connect to a listener on HOST:PORT via the TYPE protocol
	connection, err := net.Dial(TYPE, SERVER_ROUTER)

	if err != nil {
		fmt.Println("Failed to create connection to the server. Is the server listening?")
		os.Exit(1)
	}

	//Defer closing the connection to the remote listener until this function's scope closes
	defer connection.Close()

	serverConnectionDetails := connection.RemoteAddr()

	connectionIdStr := serverConnectionDetails.Network() + "://" + serverConnectionDetails.String()

	fmt.Println("Connected to: " + connectionIdStr)

	//Create a buffer to interface with the os.Stdin InputStream
	reader := bufio.NewScanner(os.Stdin)

	//Create a buffer to interface with the remote connection
	connReader := bufio.NewScanner(connection)

	fmt.Fprintf(connection, "CLIENT\n")

	connReader.Scan()
	connReader.Text()

	fmt.Println("Connected to ServerRouter")

	//Essentially a while(true) loop
	for {
		//Print out a prompt to the client
		fmt.Print("Text to Send: ")

		//Block until the enter key is pressed, then read any new content into <text>
		reader.Scan()
		text := reader.Text()

		//Trim the "newline" character from the read text
		text = strings.Trim(text, "\n")

		//Only handle the text is the text isn't empty
		if len(text) > 0 {
			fmt.Println("Sent to [" + connectionIdStr + "]: " + text)

			//Use the Fprintf to send the inputted text to the remote connection
			connection.Write([]byte(text + "\n"))

			if text != "EXIT" {
				//Block until a newline character is received from the connection
				connReader.Scan()
				message := connReader.Text()

				//Print out the response to the console
				fmt.Println("Received from [" + connectionIdStr + "]: " + message)
			} else {
				break
			}
		}
	}
}
