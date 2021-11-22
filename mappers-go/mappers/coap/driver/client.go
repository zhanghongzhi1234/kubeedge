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

package driver

import (
	"errors"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/coap/driver/coap"
	"github.com/kubeedge/mappers-go/mappers/common"
)

// CoapTCP is the configurations of coap TCP.
type CoapConfig struct {
	ServerAddress string `json:"server,omitempty"`
	//Path          string `json:"path,omitempty"`
}

// CoapClient is the structure for coap client.
type CoapClient struct {
	Client *coap.Conn
	//Handler interface{}
	Config interface{}
	//Path   string `json:"path,omitempty"`

	mu sync.Mutex
}

var clients map[string]*CoapClient

func newCoapClient(config CoapConfig) (*CoapClient, error) {
	addr := config.ServerAddress
	var coapClient *coap.Conn
	var err error

	if client, ok := clients[addr]; ok {
		return client, nil
	}

	if clients == nil {
		clients = make(map[string]*CoapClient)
	}

	//coapClient, err = coap.Dial("udp", "localhost:5683")
	coapClient, err = coap.Dial("udp", addr)
	if err != nil {
		//log.Fatalf("Error dialing: %v", err)
		klog.Fatal(err)
	}

	client := CoapClient{Client: coapClient, Config: config} //, Path: config.Path}
	clients[addr] = &client
	return &client, err
}

// NewClient allocate and return a coap client.
// Client type includes UDP only
func NewClient(config interface{}) (*CoapClient, error) {
	if coapConfig, ok := config.(CoapConfig); ok {
		return newCoapClient(coapConfig)
	} else {
		return &CoapClient{}, errors.New("wrong coap type")
	}
}

// GetStatus get device status.
// Coap don't know status, so always return good
func (c *CoapClient) GetStatus() string {
	return common.DEVSTOK
	/*c.mu.Lock()
	defer c.mu.Unlock()

	//err := c.Client.Connect()
	isConnected := c.Client.Client.IsConnected()

	if isConnected == true {
		return common.DEVSTOK
	}
	return common.DEVSTDISCONN*/
}

// Get caop value by path
func (c *CoapClient) Get(path string) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.GET,
		MessageID: 12345,
		Payload:   []byte("Get Request!"),
	}

	req.SetOption(coap.ETag, "weetag")
	req.SetOption(coap.MaxAge, 3)
	req.SetPathString(path)

	rv, err := c.Client.Send(req)
	if err != nil {
		klog.Errorf("Error sending request: %v", err)
		return nil, errors.New("error sending get request")
	}

	if rv != nil {
		klog.V(2).Info("Response payload: %s", rv.Payload)
		return rv.Payload, err
	}

	return nil, errors.New("error sending get request")
}

// Set coap value by path.
func (c *CoapClient) Set(path string, value string) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: 12345,
		Payload:   []byte(value),
	}

	req.SetOption(coap.ETag, "weetag")
	req.SetOption(coap.MaxAge, 3)
	req.SetPathString(path)

	rv, err := c.Client.Send(req)
	if err != nil {
		klog.Errorf("Error set value: %v", err)
		return nil, errors.New("error sending post request")
	}

	if rv != nil {
		klog.V(1).Info("Set result:", err, rv.Payload)
		return rv.Payload, err
	}

	return nil, errors.New("no response after sending post request")
}
