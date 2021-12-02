# kubeedge-mqtt-mapper

Kubeedge mqtt mapper use mqtt protocol to connect devices with edge node. Kubeedge also use mqtt protocol to sync data between edge node and master node. So the data flow chart is:

Master Node <----mqtt----> Edge Node <----mqtt----> Device

## Prerequisite

1. One master node and several edge node, OS is Centos 7.0, I think other linux version also ok, I use centos 7.0 only

2. All node install docker service and golang

## Install

1. Install Kubeedge cluster First, if already installed, skip this step
    + Install k8s master node
        - Add k8s yum source, please refer to official document
        - $ yum install -y kubelet-1.21.0 kubeadm-1.21.0 kubectl-1.21.0 --disableexcludes=kubernetes
        - $ systemctl enable kubelet
        - kubeadm init \
  --apiserver-advertise-address=192.168.18.26 \
  --kubernetes-version v1.21.0 \
  --service-cidr=10.96.0.0/12 \
  --pod-network-cidr=10.244.0.0/16
        - Your Kubernetes control-plane has initialized successfully, remember the token in flash screen.
    + Install kubeedge cloud service on k8s master node 
        - $ keadm init
    + Join edge node to kubeedge cluster. Do no install kubelet on edge node, it is only a nomral machine with docker service and golang
        - First execute below command in master node
        - $ keadm gettoken
        - remember the token get, then execute below command in edge node, change ip to your master node ip, and replace the token string with your token string:
        - $ keadm join --cloudcore-ipport=192.168.18.26:10000 --token=bc893748278d4bf06bcffc3430be98c4dde2eb37f5dbb66c1175624f56044a96.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzM3NDU4NzB9.OWDf1hvhz6Pa8LVlQjOqir_tpEhwcJE7tAbk9OIhN4g
    + User below command on master node to verify edge node join successfully or not:
        - $ kubectl get nodes -owide

2. Build mqtt mapper docker images from the source code, or you can pull the docker images from dockerhub: zhanghongzhi1234/mqtt-mapper
    + Upload all source code to build machine, I use one of edge node as build machine
    + $ go mod tidy
    + $ make mapper direct build only
    + Then you will see binary build successfully in bin folder, user below command to build docker images
    + $ docker build -t mqtt-mapper:v1.0.4-linux .
    + then use $ docker images command to see docker images build successfully

## Usage

1. Modify sample yaml file in deploy folder, include devicemodel, device and deployment: 

**mqtt-device-model.yaml** config properties:

```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: DeviceModel
metadata:
 name: mqtt-sample-model
 namespace: default
spec:
 properties:
  - name: temperature
    description: temperature in degree celsius
    type:
     int:
      accessMode: ReadWrite
      maximum: 100
      unit: degree celsius
  - name: temperature-enable
    description: enable data collection of temperature sensor
    type:
      int:
        accessMode: ReadWrite
        defaultValue: 1
```

**mqtt-device-instance.yaml** config 
> server address: current is tcp://127.168.20.17:1883, modify according  your mqtt broker address 

> node name: current is edge120, modify according your edge node hostname

> pathField plus inputTopic and outputTopic, combine topic to read/write from terminal device, current inputTopic is mqtt/input/device/%s/delta, and temperature property pathFile is temperature, so docker images will subscribe topic mqtt/input/device/temperature/delta to read temperature property from device

> twins field define which properties need synchronize from cloud to edge, properties not defined in twins field will not be synchronized

> desired value will be write to terminal device until success, in this example, use topic mqtt/output/device/temperature-enable/delta topic to write desire temperature-enable property value

```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: Device
metadata:
  name: mqtt-device
  labels:
    description: TISimplelinkSensorTag
    manufacturer: TexasInstruments
    model: CC2650
spec:
  deviceModelRef:
    name: mqtt-sample-model
  protocol:
    customizedProtocol:
      protocolName: mqtt
      configData:
        server: tcp://127.168.20.17:1883
        username: ""
        password: ""
        certification: ""
        inputTopic: mqtt/input/device/%s/delta
        outputTopic: mqtt/output/device/%s/delta
  nodeSelector:
    nodeSelectorTerms:
    - matchExpressions:
      - key: ''
        operator: In
        values:
        - edge120
  propertyVisitors:
    - propertyName: temperature
      customizedProtocol:
        protocolName: mqtt
        configData:
            topicField: temperature
      collectCycle: 5000
    - propertyName: temperature-enable
      customizedProtocol:
        protocolName: mqtt
        configData:
            topicField: temperature-enable
status:
  twins:
    - propertyName: temperature-enable
      reported:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: "0"
      desired:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: "0"
    - propertyName: temperature
      reported:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: "0"
      desired:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: "0"
```

2. Apply device and devicemodel yaml file on master node
    + $ kubectl apply -f mqtt-device-model.yaml
    + check result: $ kubectl get devicemodel
    + $ kubectl apply -f mqtt-device-instance.yaml
    + check relust: $ kubectl get device
    + Above command will create configmap automatically, check configmap:
    + $ kubectl get cm

3. Config deployment.yaml, modify image to your docker name:tag build in before, if download from dockerhub, no need modify; modify nodename to your edge node hostname, modify configmap name to the result when using *$ kubectl get cm* command

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mqtt-mapper
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mqttmapper
  template:
    metadata:
      labels:
        app: mqttmapper
    spec:
      hostNetwork: true
      containers:
      - name: mqtt-mapper-container
        image: mqtt-mapper:v1.0.4-linux
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
        volumeMounts:
        - name: config-volume
          mountPath: /opt/kubeedge/
        - mountPath: /dev/ttyS0
          name: mqtt-dev0
        - mountPath: /dev/ttyS1
          name: mqtt-dev1
      nodeName: edge120
      volumes:
      - name: config-volume
        configMap:
          name: device-profile-config-edge120
      - name: mqtt-dev0
        hostPath:
          path: /dev/ttyS0
      - name: mqtt-dev1
        hostPath:
          path: /dev/ttyS1
      restartPolicy: Always
```
4. Apply deployment.yaml on master node to start all
    + $ kubectl apply -f deployment.yaml
    + check result: 
    + $ kubectl get deploy
    + $ kubectl get pod
    + You will see both deploy and pod change status to running, also check on edge node:
    + $ docker ps
    + you will see mqtt-mapper docker image start automatically
5. Send data to device input topic to simulate terminal data update, my mqtt broker is mosquitto, so I just use its commandline:
    + mosquitto_pub -h localhost -t mqtt/input/device/mqtt-device/delta -m '{"temperature":"26","temperature-enable":"11"}'
6. Check if data read successfully:
    + execute below command on edge node to view if docker image successfully read the data from terminal:
    + $ docker logs –f \<containerid\> 
    + you will see temperature updated as 26 and temperatue-enable become 11 in the scroll screen in edge node
    + execute below command on master node to view if cloud side successfully get the data, should be realtime update:
    + $ kubectl get devices xxxxx -o yaml –w
    + you will see temperature updated as 26 and temperatue-enable become 11 in the scroll screen in master node too
7. Test write: change mqtt-device-instance.yaml desired value, then apply again, you will see docker logs, send new desired value to output topic, using mqtt client such as mqttfx to subscribe this topic to see if write successfully or not. In real environment, we usually use k8s api to perform write function, and it can be combined with front end dashboard.


## Contributing

PRs accepted.

## License

KubeEdge is under the Apache 2.0 license. See the [LICENSE](../../LICENSE) file for details..