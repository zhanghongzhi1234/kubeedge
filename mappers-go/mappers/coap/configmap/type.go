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

// CoapVisitorConfig is the coap register configuration.
type CoapVisitorConfig struct {
	ProtocolName      string `json:"protocolName"`
	VisitorConfigData `json:"configData"`
}

type VisitorConfigData struct {
	PathField string `json:"pathField,omitempty"`
}

// CoapProtocolConfig is the protocol configuration.
type CoapProtocolConfig struct {
	//SlaveID        int16      `json:"slaveID,omitempty"`
	ProtocolName   string     `json:"protocolName"`
	CoapConfigData ConfigData `json:"configData"`
}

type ConfigData struct {
	ServerAddress string `json:"server,omitempty"`
	/*Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Cert          string `json:"certification,omitempty"`*/
	//Path string `json:"path,omitempty"`
}

// CoapProtocolCommonConfig is the coap protocol configuration.
type CoapProtocolCommonConfig struct {
	CustomizedValues CustomizedValue `json:"customizedValues,omitempty"`
}

// CustomizedValue is the customized part for coap protocol.
type CustomizedValue map[string]interface{}
