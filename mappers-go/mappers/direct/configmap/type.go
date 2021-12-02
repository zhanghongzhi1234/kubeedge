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

package configmap

// DirectVisitorConfig is the direct register configuration.
type DirectVisitorConfig struct {
	ProtocolName      string `json:"protocolName"`
	VisitorConfigData `json:"configData"`
}

type VisitorConfigData struct {
	TopicField string `json:"topicField,omitempty"`
}

// DirectProtocolConfig is the protocol configuration.
type DirectProtocolConfig struct {
	//SlaveID        int16      `json:"slaveID,omitempty"`
	ProtocolName   string     `json:"protocolName"`
	MQTTConfigData ConfigData `json:"configData"`
}

type ConfigData struct {
	ServerAddress string `json:"server,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Cert          string `json:"certification,omitempty"`
	InputTopic    string `json:"inputTopic,omitempty"`
	OutputTopic   string `json:"outputTopic,omitempty"`
}

// DirectProtocolCommonConfig is the direct protocol configuration.
type DirectProtocolCommonConfig struct {
	CustomizedValues CustomizedValue `json:"customizedValues,omitempty"`
}

// CustomizedValue is the customized part for direct protocol.
type CustomizedValue map[string]interface{}
