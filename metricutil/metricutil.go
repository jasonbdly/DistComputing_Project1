package metricutil

import (
	"strings"
	"os"
	"path/filepath"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

var metrics map[string]int64 = make(map[string]int64)

type metricData struct {
	name string
	val int64
	metrictype string
}

var metricInput chan metricData = make(chan metricData, 50)
var metricsRead chan bool = make(chan bool, 1)

/*func main() {
	Start()

	AddVal("TEST", 1)
	AddVal("TEST", 1)
	AddVal("TEST", 1)
	AddVal("TEST", 5)
	AddVal("TEST", 2)

	SetVal("TEST_SET", 100)
	SetVal("TEST_SET2", 500)
	SetVal("TEST_SET3", 1000000000)

	Stop("../test_metrics/test_metric.json")
}*/

func Start() {
	go func() {
		for {
			nextMetric, hasMore := <- metricInput

			if nextMetric.metrictype == "set" {
				metrics[nextMetric.name] = nextMetric.val
			} else if nextMetric.metrictype == "add" {
				metrics[nextMetric.name] += nextMetric.val
				metrics[nextMetric.name + "_NUM"]++
			} else if nextMetric.metrictype == "setnumberofitems" {
				metrics[nextMetric.name + "_NUM"] = nextMetric.val
			}

			if !hasMore {
				break
			}
		}

		metricsRead <- true
	}()
}

func Stop(location string) {
	close(metricInput)

	<- metricsRead

	fileData := make(map[string]int64)

	for key, value := range metrics {
		if !strings.HasSuffix(key, "_NUM") {
			if metrics[key + "_NUM"] > 0 {
				fileData[key + "_AVG"] = value / metrics[key + "_NUM"]
			} else {
				fileData[key + "_TOTAL"] = value
			}
		}
	}

	fileDataJSON, _ := json.Marshal(fileData)

	fmt.Println(string(fileDataJSON))

	locationParts := strings.Split(location, "/")

	//Ensure the full folder path exists
	os.MkdirAll(filepath.Join(locationParts[:len(locationParts) - 1]...), os.ModePerm)

	location = filepath.Join(strings.Split(location, "/")...)

	fileLocation, err := filepath.Abs(location)
	if err != nil {
		fmt.Println("Failed to resolve metrics file path: " + err.Error())
	}

	err = ioutil.WriteFile(fileLocation, fileDataJSON, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to write metrics to file: " + err.Error())
	}
}

func SetVal(valName string, val int64) {
	metricInput <- metricData{valName, val, "set"}
}

func SetNumberOfItems(valName string, val int64) {
	metricInput <- metricData{valName, val, "setnumberofitems"}
}

func AddVal(valName string, val int64) {
	metricInput <- metricData{valName, val, "add"}
}