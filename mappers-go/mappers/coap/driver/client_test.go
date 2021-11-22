/*
Copyright 2020 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This application needs physical devices.
// Please edit by demand for testing.

package driver

import (
	"fmt"
	"os"
	"time"
)

func tdriver() {
	var coaprtu CoapRTU

	coaprtu.SerialName = "/dev/ttyS0"
	coaprtu.BaudRate = 9600
	coaprtu.DataBits = 8
	coaprtu.StopBits = 1
	coaprtu.SlaveID = 1
	coaprtu.Parity = "N"
	coaprtu.Timeout = 2 * time.Second

	client, err := NewClient(coaprtu)
	if err != nil {
		fmt.Println("New client error")
		os.Exit(1)
	}

	results, err := client.Set("DiscreteInputRegister", 2, 1)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(results)
	results, err = client.Set("CoilRegister", 2, 1)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(results)
	os.Exit(0)
}

func main() {
	/*
		var coaptcp CoapTCP

		coaptcp.DeviceIp = "192.168.56.1"
		coaptcp.TcpPort = "502"
		coaptcp.SlaveId = 0x1
		client := NewClient(coaptcp)
		if client == nil {
			fmt.Println("New client error")
			os.Exit(1)
		}
		fmt.Println("status: ", client.GetStatus())

		results, err := client.Client.ReadDiscreteInputs(0, 1)
		if err != nil {
			fmt.Println("Read error: ", err)
			os.Exit(1)
		}
		fmt.Println("result: ", results)
	*/
	tdriver()
	os.Exit(0)
}
