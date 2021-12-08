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

	"github.com/kubeedge/mappers-go/mappers/coap/configmap"
	"github.com/kubeedge/mappers-go/mappers/coap/driver"
	"github.com/kubeedge/mappers-go/mappers/coap/globals"
	"github.com/kubeedge/mappers-go/mappers/common"
)

var devices map[string]*globals.CoapDev
var models map[string]common.DeviceModel
var protocols map[string]common.Protocol
var wg sync.WaitGroup

// DeviceTwinDelta twin delta.
/*type MqttTwinUpdate struct {
	Delta map[string]string `json:"update"`
}*/

// setVisitor check if visitory property is readonly, if not then set it.
func setVisitor(visitorConfig *configmap.CoapVisitorConfig, twin *common.Twin, client *driver.CoapClient) {
	if twin.PVisitor.PProperty.AccessMode == "ReadOnly" {
		klog.V(1).Info("Visit readonly property: ", visitorConfig.VisitorConfigData.PathField)
		return
	}

	_, err := client.Set(visitorConfig.VisitorConfigData.PathField, twin.Desired.Value)
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

	var dev *globals.CoapDev
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
	klog.V(2).Infof("Receive message parsed: %v", delta)
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
		var visitorConfig configmap.CoapVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal visitor config failed: %v", err)
			continue
		}
		setVisitor(&visitorConfig, &dev.Instance.Twins[i], dev.CoapClient)
	}
}

// initCoap initialize coap client
func initCoap(protocolConfig configmap.CoapProtocolConfig, instanceID string) (client *driver.CoapClient, err error) {
	if protocolConfig.CoapConfigData.ServerAddress != "" {
		coapConfig := driver.CoapConfig{
			ServerAddress: protocolConfig.CoapConfigData.ServerAddress,
			//Path:          protocolConfig.CoapConfigData.Path,
		}
		client, err = driver.NewClient(coapConfig)

	} else {
		return nil, errors.New("no protocol found")
	}

	return client, err
}

// initTwin initialize the timer to get twin value.
func initTwin(dev *globals.CoapDev) {
	for i := 0; i < len(dev.Instance.Twins); i++ {
		var visitorConfig configmap.CoapVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal VisitorConfig error: %v", err)
			continue
		}
		setVisitor(&visitorConfig, &dev.Instance.Twins[i], dev.CoapClient)

		twinData := TwinData{Client: dev.CoapClient,
			Name:          dev.Instance.Twins[i].PropertyName,
			Type:          dev.Instance.Twins[i].Desired.Metadatas.Type,
			VisitorConfig: &visitorConfig,
			Topic:         fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID)}
		collectCycle := time.Duration(dev.Instance.Twins[i].PVisitor.CollectCycle) * time.Millisecond //time.Duration is nanosecond
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
func initData(dev *globals.CoapDev) {
	for i := 0; i < len(dev.Instance.Datas.Properties); i++ {
		var visitorConfig configmap.CoapVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Datas.Properties[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Errorf("Unmarshal VisitorConfig error: %v", err)
			continue
		}
		twinData := TwinData{Client: dev.CoapClient,
			Name:          dev.Instance.Datas.Properties[i].PropertyName,
			Type:          dev.Instance.Datas.Properties[i].Metadatas.Type,
			VisitorConfig: &visitorConfig,
			Topic:         fmt.Sprintf(common.TopicDataUpdate, dev.Instance.ID)}
		collectCycle := time.Duration(dev.Instance.Datas.Properties[i].PVisitor.CollectCycle) * time.Millisecond
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
func initGetStatus(dev *globals.CoapDev) {
	getStatus := GetStatus{Client: dev.CoapClient,
		topic: fmt.Sprintf(common.TopicStateUpdate, dev.Instance.ID)}
	timer := common.Timer{Function: getStatus.Run, Duration: 1 * time.Second, Times: 0}
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer.Start()
	}()
}

// start start the device.
func start(dev *globals.CoapDev) {
	if !strings.Contains(dev.Instance.ProtocolName, "customized-protocol-coap-device") {
		klog.Errorf("%v start fail, protocol not supported: %v", dev.Instance.ID, dev.Instance.ProtocolName)
		return
	}
	var protocolCommConfig configmap.CoapProtocolCommonConfig
	if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolCommonConfig), &protocolCommConfig); err != nil {
		klog.Errorf("Unmarshal ProtocolCommonConfig error: %v", err)
		return
	}

	var protocolConfig configmap.CoapProtocolConfig
	if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolConfigs), &protocolConfig); err != nil {
		klog.Errorf("Unmarshal ProtocolConfigs error: %v", err)
		return
	}

	//dev.Path = protocolConfig.CoapConfigData.Path //save topic by device
	client, err := initCoap(protocolConfig, dev.Instance.ID)
	if err != nil {
		klog.Errorf("Init error: %v", err)
		return
	}
	dev.CoapClient = client

	initTwin(dev)
	initData(dev)

	//if !globals.LocalTest {
	if err := initSubscribeMqtt(dev.Instance.ID); err != nil {
		klog.Errorf("Init subscribe mqtt error: %v", err)
		return
	}
	//}

	klog.V(1).Info(dev.Instance.ID, " start successfully")

	initGetStatus(dev)
}

// DevInit initialize the device datas.
func DevInit(configmapPath string) error {
	devices = make(map[string]*globals.CoapDev)
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
