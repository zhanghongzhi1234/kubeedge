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

	"encoding/json"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/common"
)

// DirectTCP is the configurations of direct TCP.
type DirectConfig struct {
	ServerAddress string `json:"server,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Cert          string `json:"certification,omitempty"`
	Topic         string `json:"topic,omitempty"`
}

// DirectClient is the structure for direct client.
type DirectClient struct {
	//Client  modbus.Client
	Client common.MqttClient
	//Handler interface{}
	Config interface{}
	Topic  string `json:"topic,omitempty"`

	mu sync.Mutex
}

var clients map[string]*DirectClient

func newMQTTClient(config DirectConfig) (*DirectClient, error) {
	addr := config.ServerAddress
	var mqttClent common.MqttClient
	var err error

	if client, ok := clients[addr]; ok {
		return client, nil
	}

	if clients == nil {
		clients = make(map[string]*DirectClient)
	}

	//mqttClent = common.MqttClient{IP: "tcp://127.0.0.1:1883",
	mqttClent = common.MqttClient{IP: config.ServerAddress,
		User:       config.Username,
		Passwd:     config.Password,
		Cert:       config.Cert,
		PrivateKey: ""}
	if err = mqttClent.Connect(); err != nil {
		klog.Fatal(err)
	}

	client := DirectClient{Client: mqttClent, Config: config, Topic: config.Topic}
	clients[addr] = &client
	return &client, err
}

// NewClient allocate and return a direct client.
// Client type includes TCP and RTU.
func NewClient(config interface{}) (*DirectClient, error) {
	if directConfig, ok := config.(DirectConfig); ok {
		return newMQTTClient(directConfig)
	} else {
		return &DirectClient{}, errors.New("Wrong direct type")
	}
}

// GetStatus get device status.
// Now we could only get the connection status.
func (c *DirectClient) GetStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	//err := c.Client.Connect()
	isConnected := c.Client.Client.IsConnected()

	if isConnected == true {
		return common.DEVSTOK
	}
	return common.DEVSTDISCONN
}

// Get get register.
/*func (c *DirectClient) Get(registerType string, addr uint16, quantity uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	klog.V(2).Info("Get result: ", results)
	return results, err
}*/

// Set set register.
func (c *DirectClient) Set(topicField string, value string) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	klog.V(1).Infof("Set %v to %v", topicField, value)

	topicValueMap := map[string]string{topicField: value}
	results, _ = json.Marshal(topicValueMap)

	if directConfig, ok := c.Config.(DirectConfig); ok {
		if err = c.Client.Publish(directConfig.Topic, results); err != nil {
			klog.Errorf("Publish topic %v failed, err: %v", directConfig.Topic, err)
		} else {
			klog.V(1).Infof("Publish topic %v successfully, value: %v", directConfig.Topic, topicValueMap)
		}
	}

	klog.V(1).Info("Set result:", err, value)
	return results, err
}
