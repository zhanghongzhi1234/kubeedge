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

package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/common"
	"github.com/kubeedge/mappers-go/mappers/direct/configmap"
	"github.com/kubeedge/mappers-go/mappers/direct/driver"
	"github.com/kubeedge/mappers-go/mappers/direct/globals"
)

var devices map[string]*globals.DirectDev
var models map[string]common.DeviceModel
var protocols map[string]common.Protocol
var wg sync.WaitGroup

// DeviceTwinDelta twin delta.
/*type MqttTwinUpdate struct {
	Delta map[string]string `json:"update"`
}*/

// setVisitor check if visitory property is readonly, if not then set it.
func setVisitor(visitorConfig *configmap.DirectVisitorConfig, twin *common.Twin, client *driver.DirectClient) {
	if twin.PVisitor.PProperty.AccessMode == "ReadOnly" {
		klog.V(1).Info("Visit readonly property: ", client.Topic)
		return
	}

	_, err := client.Set(visitorConfig.VisitorConfigData.TopicField, twin.Desired.Value)
	if err != nil {
		klog.Errorf("Set visitor error: %v %v", err, visitorConfig)
		return
	}
}

// getDeviceID extract the device ID from Mqtt topic.
func getDeviceID(topic string) (id string) {
	re := regexp.MustCompile(`hw/events/device/(.+)/twin/update/delta`)
	return re.FindStringSubmatch(topic)[1]
}

// getDeviceID extract the device ID from Mqtt topic.
func getTwinDeviceID(topic string) (id string) {
	re := regexp.MustCompile(`mqtt/input/device/(.+)/delta`)
	return re.FindStringSubmatch(topic)[1]
}

// onMessage callback function of Mqtt subscribe message.
func onMessage(client mqtt.Client, message mqtt.Message) {
	klog.V(2).Info("Receive message", message.Topic())
	// Get device ID and get device instance
	id := getDeviceID(message.Topic())
	if id == "" {
		klog.Error("Wrong topic")
		return
	}
	klog.V(2).Info("Device id: ", id)

	var dev *globals.DirectDev
	var ok bool
	if dev, ok = devices[id]; !ok {
		klog.Error("Device not exist")
		return
	}

	// Get twin map key as the propertyName
	var delta common.DeviceTwinDelta
	if err := json.Unmarshal(message.Payload(), &delta); err != nil {
		klog.Errorf("Unmarshal message failed: %v", err)
		return
	}
	for twinName, twinValue := range delta.Delta {
		i := 0
		for i = 0; i < len(dev.Instance.Twins); i++ {
			if twinName == dev.Instance.Twins[i].PropertyName {
				break
			}
		}
		if i == len(dev.Instance.Twins) {
			klog.Error("Twin not found: ", twinName)
			continue
		}
		// Desired value is not changed.
		if dev.Instance.Twins[i].Desired.Value == twinValue {
			continue
		}
		dev.Instance.Twins[i].Desired.Value = twinValue
		var visitorConfig configmap.DirectVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal visitor config failed: %v", err)
			continue
		}
		setVisitor(&visitorConfig, &dev.Instance.Twins[i], dev.DirectClient)
	}
}

// onMessage callback function of Mqtt subscribe message.
func onTwinMessage(client mqtt.Client, message mqtt.Message) {
	var err error
	klog.V(1).Info("Receive message", message.Topic())
	// Get device ID and get device instance
	id := getTwinDeviceID(message.Topic())
	if id == "" {
		klog.Error("Wrong topic")
		return
	}
	klog.V(1).Info("Device id: ", id)

	var dev *globals.DirectDev
	var ok bool
	if dev, ok = devices[id]; !ok {
		klog.Error("Device not exist")
		return
	}

	var delta map[string]string
	if err := json.Unmarshal(message.Payload(), &delta); err != nil {
		klog.Errorf("Unmarshal message failed: %v", err)
		return
	}
	// Get twin map key as the propertyName
	/*var delta MqttTwinUpdate
	if err := json.Unmarshal(message.Payload(), &delta); err != nil {
		klog.Errorf("Unmarshal message failed: %v", err)
		return
	}*/
	for twinName, twinValue := range delta {
		i := 0
		//dev.DirectClient.Set(name, value)

		for i = 0; i < len(dev.Instance.Twins); i++ {
			if twinName == dev.Instance.Twins[i].PropertyName {
				break
			}
		}
		if i == len(dev.Instance.Twins) {
			klog.Error("Twin not found: ", twinName)
			continue
		}
		topic := fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID)
		// construct payload
		var payload []byte
		if payload, err = common.CreateMessageTwinUpdate(twinName, "string", twinValue); err != nil {
			klog.Error("Create message twin update failed")
			return
		}

		if err := globals.MqttClient.Publish(topic, payload); err != nil {
			klog.Errorf("Publish topic %v failed, err: %v", topic, err)
		}
		klog.V(1).Infof("Update the %s value as %s", twinName, twinValue)
	}
}

// initDirect initialize direct client
func initDirect(protocolConfig configmap.DirectProtocolConfig, instanceID string) (client *driver.DirectClient, err error) {
	if protocolConfig.MQTTConfigData.ServerAddress != "" {
		directConfig := driver.DirectConfig{
			ServerAddress: protocolConfig.MQTTConfigData.ServerAddress,
			Username:      protocolConfig.MQTTConfigData.Username,
			Password:      protocolConfig.MQTTConfigData.Password,
			Cert:          protocolConfig.MQTTConfigData.Cert,
			Topic:         fmt.Sprintf(protocolConfig.MQTTConfigData.OutputTopic, instanceID)}
		client, err = driver.NewClient(directConfig)

	} else {
		return nil, errors.New("No protocol found")
	}

	return client, nil
}

// initTwin initialize the timer to get twin value.
/*func initTwin(dev *globals.DirectDev) {
	for i := 0; i < len(dev.Instance.Twins); i++ {
		var visitorConfig configmap.DirectVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal VisitorConfig error: %v", err)
			continue
		}
		setVisitor(&visitorConfig, &dev.Instance.Twins[i], dev.DirectClient)

		twinData := TwinData{Client: dev.DirectClient,
			Name:          dev.Instance.Twins[i].PropertyName,
			Type:          dev.Instance.Twins[i].Desired.Metadatas.Type,
			VisitorConfig: &visitorConfig,
			Topic:         fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID)}
		collectCycle := time.Duration(dev.Instance.Twins[i].PVisitor.CollectCycle)
		// If the collect cycle is not set, set it to 1 second.
		if collectCycle == 0 {
			collectCycle = 1 * time.Second
		}
		timer := common.Timer{Function: twinData.Run, Duration: collectCycle, Times: 0}
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer.Start()
		}()
	}
}

// initData initialize the timer to get data.
func initData(dev *globals.DirectDev) {
	for i := 0; i < len(dev.Instance.Datas.Properties); i++ {
		var visitorConfig configmap.DirectVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Datas.Properties[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal VisitorConfig error: %v", err)
			continue
		}
		twinData := TwinData{Client: dev.DirectClient,
			Name:          dev.Instance.Datas.Properties[i].PropertyName,
			Type:          dev.Instance.Datas.Properties[i].Metadatas.Type,
			VisitorConfig: &visitorConfig,
			Topic:         fmt.Sprintf(common.TopicDataUpdate, dev.Instance.ID)}
		collectCycle := time.Duration(dev.Instance.Datas.Properties[i].PVisitor.CollectCycle)
		// If the collect cycle is not set, set it to 1 second.
		if collectCycle == 0 {
			collectCycle = 1 * time.Second
		}
		timer := common.Timer{Function: twinData.Run, Duration: collectCycle, Times: 0}
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer.Start()
		}()
	}
}*/

// initTwinMqtt subscribe Mqtt topics from outer device.
func initTwinMqtt(deviceTopic string, instanceID string) error {
	topic := fmt.Sprintf(deviceTopic, instanceID)
	klog.V(1).Info("Subscribe topic: ", topic)
	return globals.MqttClient.Subscribe(topic, onTwinMessage)
}

// initSubscribeMqtt subscribe Mqtt topics from cloudcore.
func initSubscribeMqtt(instanceID string) error {
	var topic string
	if globals.LocalTest {
		topic = fmt.Sprintf("hw/events/device/%s/twin/update/delta", instanceID)
	} else {
		topic = fmt.Sprintf(common.TopicTwinUpdateDelta, instanceID)
	}
	klog.V(1).Info("Subscribe topic: ", topic)
	return globals.MqttClient.Subscribe(topic, onMessage)
}

// initGetStatus start timer to get device status and send to eventbus.
func initGetStatus(dev *globals.DirectDev) {
	getStatus := GetStatus{Client: dev.DirectClient,
		topic: fmt.Sprintf(common.TopicStateUpdate, dev.Instance.ID)}
	timer := common.Timer{Function: getStatus.Run, Duration: 1 * time.Second, Times: 0}
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer.Start()
	}()
}

// start start the device.
func start(dev *globals.DirectDev) {
	if strings.Contains(dev.Instance.ProtocolName, "customized-protocol-mqtt-device") == false {
		klog.Errorf("%v start fail, protocol not supported: %v", dev.Instance.ID, dev.Instance.ProtocolName)
		return
	}
	var protocolCommConfig configmap.DirectProtocolCommonConfig
	if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolCommonConfig), &protocolCommConfig); err != nil {
		klog.Errorf("Unmarshal ProtocolCommonConfig error: %v", err)
		return
	}

	var protocolConfig configmap.DirectProtocolConfig
	if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolConfigs), &protocolConfig); err != nil {
		klog.Errorf("Unmarshal ProtocolConfigs error: %v", err)
		return
	}

	dev.Topic = protocolConfig.MQTTConfigData.InputTopic //save topic by device
	//client, err := initDirect(protocolCommConfig, protocolConfig.SlaveID)
	client, err := initDirect(protocolConfig, dev.Instance.ID)
	if err != nil {
		klog.Errorf("Init error: %v", err)
		return
	}
	dev.DirectClient = client

	//initTwin(dev)
	//initData(dev)

	if err := initTwinMqtt(dev.Topic, dev.Instance.ID); err != nil {
		klog.Errorf("Init subscribe mqtt error: %v", err)
		return
	}

	if err := initSubscribeMqtt(dev.Instance.ID); err != nil {
		klog.Errorf("Init subscribe mqtt error: %v", err)
		return
	}

	klog.V(1).Info(dev.Instance.ID, " start successfully")

	initGetStatus(dev)
}

// DevInit initialize the device datas.
func DevInit(configmapPath string) error {
	devices = make(map[string]*globals.DirectDev)
	models = make(map[string]common.DeviceModel)
	protocols = make(map[string]common.Protocol)
	return configmap.Parse(configmapPath, devices, models, protocols)
}

// DevStart start all devices.
func DevStart() {
	for id, dev := range devices {
		klog.V(4).Info("Dev: ", id, dev)
		start(dev)
	}
	wg.Wait()
}
