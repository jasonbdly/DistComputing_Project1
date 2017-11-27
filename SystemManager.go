package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

const (
	startPort           = 30000
	testFile            = "testfile.txt"
	serverRouterExe_W   = "ServerRouter.exe"
	serverRouterExe_NIX = "./ServerRouter"
	p2pExe_W            = "P2P.exe"
	p2pExe_NIX          = "./P2P"
)

func main() {
	var serverRouterExe string
	var p2pExe string

	if runtime.GOOS == "windows" {
		serverRouterExe = serverRouterExe_W
		p2pExe = p2pExe_W
	} else {
		serverRouterExe = serverRouterExe_NIX
		p2pExe = p2pExe_NIX
	}

	fmt.Println("How many nodes should be created?")
	inputReader := bufio.NewScanner(os.Stdin)
	inputReader.Scan()

	numNodesSTR := inputReader.Text()
	if len(numNodesSTR) == 0 {
		numNodesSTR = "100"
	}

	numNodes, err := strconv.Atoi(numNodesSTR)
	if err != nil {
		fmt.Println("Invalid number of nodes. Defaulting to 100")
		numNodes = 100
	}

	serverRouters := []string{"localhost:6001", "localhost:6000"}

	fmt.Println("Starting server routers")

	serverRouter1 := exec.Command(serverRouterExe, "", "6000", serverRouters[0])
	serverRouter1.Stdout = os.Stdout
	serverRouter1.Stderr = os.Stderr
	err = serverRouter1.Start()
	if err != nil {
		fmt.Println("Failed to start server router 1: " + err.Error())
		return
	}

	serverRouter2 := exec.Command(serverRouterExe, "", "6001", serverRouters[1])
	serverRouter2.Stdout = os.Stdout
	serverRouter2.Stderr = os.Stderr
	err = serverRouter2.Start()
	if err != nil {
		fmt.Println("Failed to start server router 2: " + err.Error())
		return
	}

	p2pNodes := []*exec.Cmd{}
	for i := 0; i < numNodes; i++ {
		fmt.Println("Starting P2P Node: " + strconv.Itoa(i))
		p2pNode := exec.Command(p2pExe, "", strconv.Itoa(startPort), strconv.Itoa(startPort+numNodes), serverRouters[i%len(serverRouters)], testFile, strconv.Itoa(20*numNodes))
		p2pNode.Stdout = os.Stdout
		p2pNode.Stderr = os.Stderr
		err = p2pNode.Start()
		if err != nil {
			fmt.Println("Failed to start P2P node: " + strconv.Itoa(i))
		}
		p2pNodes = append(p2pNodes, p2pNode)
	}

	//Wait for all P2P nodes to finish
	for _, p2pNode := range p2pNodes {
		p2pNode.Wait()
	}

	fmt.Println("All P2P nodes finished. Shutting down server routers.")

	err = serverRouter1.Process.Signal(os.Kill)
	if err != nil {
		fmt.Println("Failed to stop server router 1: " + err.Error())
	}

	err = serverRouter2.Process.Signal(os.Kill)
	if err != nil {
		fmt.Println("Failed to stop server router 2: " + err.Error())
	}

	fmt.Println("All processes complete.")
}
