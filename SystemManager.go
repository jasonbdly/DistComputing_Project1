package main

import (
	metrics "./metricutil"
	p2p "./p2pmessage"
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	startPort           = 49154
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

	//First, clear any older p2p metric results
	os.RemoveAll(filepath.Join(strings.Split("./metrics/p2p", "/")...))

	metrics.Start()

	inputReader := bufio.NewScanner(os.Stdin)

	fmt.Println("Do you want to run both server routers on this machine?")
	inputReader.Scan()
	runBothSRsAnswer := inputReader.Text()
	runBothSRs := strings.ToLower(runBothSRsAnswer) != "no" && strings.ToLower(runBothSRsAnswer) != "n"

	fmt.Println("How many nodes should be created?")
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

	serverRouters := []string{"localhost:49153", "localhost:49152"}

	var otherSRAddress string
	if !runBothSRs {
		fmt.Println("=== The server router's address is: (" + p2p.GetLANAddress() + "49152) ===")

		fmt.Println("Enter the other server router's full address (host:port): ")
		inputReader.Scan()

		otherSRAddress = inputReader.Text()
	} else {
		otherSRAddress = serverRouters[0]
	}

	fmt.Println("Starting server router 1")

	sr1StartTime := time.Now()

	serverRouter1 := exec.Command(serverRouterExe, "", "49152", otherSRAddress)
	//serverRouter1.Stdout = os.Stdout
	serverRouter1.Stderr = os.Stderr
	err = serverRouter1.Start()
	if err != nil {
		fmt.Println("Failed to start server router 1: " + err.Error())
		return
	}

	metrics.AddVal("ServerRouter_Startup_Time_NS", int64(time.Since(sr1StartTime)))

	var serverRouter2 *exec.Cmd
	var sr2StartTime time.Time
	if runBothSRs {
		fmt.Println("Starting server router 2")

		sr2StartTime = time.Now()

		serverRouter2 = exec.Command(serverRouterExe, "", "49153", serverRouters[1])
		//serverRouter2.Stdout = os.Stdout
		serverRouter2.Stderr = os.Stderr
		err = serverRouter2.Start()
		if err != nil {
			fmt.Println("Failed to start server router 2: " + err.Error())
			return
		}

		metrics.AddVal("ServerRouter_Startup_Time_NS", int64(time.Since(sr2StartTime)))
	}

	nodeStartTimes := []time.Time{}
	p2pNodes := []*exec.Cmd{}
	for i := 0; i < numNodes; i++ {
		fmt.Println("Starting P2P Node " + strconv.Itoa(i))
		nodeStartTimes = append(nodeStartTimes, time.Now())

		var srAddress string
		if runBothSRs {
			srAddress = serverRouters[i%len(serverRouters)]
		} else {
			srAddress = otherSRAddress
		}

		p2pNode := exec.Command(p2pExe, "", strconv.Itoa(startPort+i), srAddress, testFile, strconv.Itoa(10*numNodes))
		p2pNode.Stdout = os.Stdout
		p2pNode.Stderr = os.Stderr
		err = p2pNode.Start()
		if err != nil {
			fmt.Println("Failed to start P2P node: " + strconv.Itoa(i))
		}

		metrics.AddVal("P2P_Node_Startup_Time_NS", int64(time.Since(nodeStartTimes[i])))

		p2pNodes = append(p2pNodes, p2pNode)
	}

	//Wait for all P2P nodes to finish
	for i, p2pNode := range p2pNodes {
		fmt.Println("Waiting on P2P Node " + strconv.Itoa(i))
		p2pNode.Wait()

		metrics.AddVal("P2P_Node_Running_Time_NS", int64(time.Since(nodeStartTimes[i])))
	}

	fmt.Println("All P2P nodes finished. Shutting down server routers.")

	err = serverRouter1.Process.Signal(os.Kill)
	if err != nil {
		fmt.Println("Failed to stop server router 1: " + err.Error())
	}

	metrics.AddVal("ServerRouter_Running_Time_NS", int64(time.Since(sr1StartTime)))

	fmt.Println("Server router 1 stopped")

	if runBothSRs {
		err = serverRouter2.Process.Signal(os.Kill)
		if err != nil {
			fmt.Println("Failed to stop server router 2: " + err.Error())
		}

		metrics.AddVal("ServerRouter_Running_Time_NS", int64(time.Since(sr2StartTime)))

		fmt.Println("Server router 2 stopped")
	}

	fmt.Println("Aggregating Metrics...")

	nodeMetricFiles, err := ioutil.ReadDir(filepath.Join(strings.Split("./metrics/P2P", "/")...))
	if err != nil {
		fmt.Println("Failed to retrieve P2P node metrics from file system: " + err.Error())
	}

	var nodeMetricData map[string]int64
	for _, nodeMetricFile := range nodeMetricFiles {
		nodeMetricFileData, err := ioutil.ReadFile(filepath.Join(strings.Split("./metrics/P2P/"+nodeMetricFile.Name(), "/")...))
		if err != nil {
			fmt.Println("Failed to read metric file for node " + nodeMetricFile.Name() + ": " + err.Error())
			break
		}

		err = json.Unmarshal(nodeMetricFileData, &nodeMetricData)
		if err != nil {
			fmt.Println("Failed to unmarshal metric file for node " + nodeMetricFile.Name() + ": " + err.Error())
			break
		}

		for key, value := range nodeMetricData {
			metrics.AddVal(key, value)
		}
	}

	fmt.Println("Writing Metrics to File System...")

	metrics.Stop("./metrics/SystemManager_" + strconv.Itoa(numNodes) + "_Nodes.json")

	fmt.Println("All processes complete.")
}
