# kubeedge-coap-mapper

Kubeedge coap mapper use coap protocol to connect devices with edge node. Kubeedge also use coap protocol to sync data between edge node and master node. So the data flow chart is:

Master Node <----mqtt----> Edge Node <----coap----> Device

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

2. Build coap mapper docker images from the source code, or you can pull the docker images from dockerhub: zhanghongzhi1234/coap-mapper
    + Upload all source code to build machine, I use one of edge node as build machine
    + $ go mod tidy
    + $ make mapper coap build only
    + Then you will see binary build successfully in bin folder, user below command to build docker images
    + $ docker build -t coap-mapper:v1.0.3-linux .
    + then use $ docker images command to see docker images build successfully

## Usage

1. Modify sample yaml file in deploy folder, include devicemodel, device and deployment: 

**coap-device-model.yaml** config properties:

```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: DeviceModel
metadata:
 name: coap-sample-model
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

**coap-device-instance.yaml** config 
> server address: current is 127.0.0.1:5683, modify according  your coap server address 

> node name: current is edge120, modify according your edge node hostname

> pathField used in coap protocol path field, docker images send get request to coap server attached with path to read temperature property from device

> twins field define which properties need synchronize from cloud to edge, properties not defined in twins field will not be synchronized

> desired value will be write to terminal device until success, in this example, docker images will send put reqeust to coap server to write desired property value

> collectCycle is the interval in millisecond for docker container to poll coap server to read properties value, if not set, this property will be polled in default 1 second interval

```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: Device
metadata:
  name: coap-device
  labels:
    description: TISimplelinkSensorTag
    manufacturer: TexasInstruments
    model: CC2650
spec:
  deviceModelRef:
    name: coap-sample-model
  protocol:
    customizedProtocol:
      protocolName: coap
      configData:
        server: 127.0.0.1:5683
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
        protocolName: coap
        configData:
            pathField: temperature
      collectCycle: 5000
    - propertyName: temperature-enable
      customizedProtocol:
        protocolName: coap
        configData:
            pathField: temperature/enable
status:
  twins:
    - propertyName: temperature-enable
      reported:
        metadata:
          timestamp: '1550049403598'
          type: string
        value: "0"
      desired:
        metadata:
          timestamp: '1550049403598'
          type: string
        value: "0"
    - propertyName: temperature
      reported:
        metadata:
          timestamp: '1550049403598'
          type: string
        value: "0"
      desired:
        metadata:
          timestamp: '1550049403598'
          type: string
        value: "0"
```

2. Apply device and devicemodel yaml file on master node
    + $ kubectl apply -f coap-device-model.yaml
    + check result: $ kubectl get devicemodel
    + $ kubectl apply -f coap-device-instance.yaml
    + check relust: $ kubectl get device
    + Above command will create configmap automatically, check configmap:
    + $ kubectl get cm

3. Config deployment.yaml, modify image to your docker name:tag build in before, if download from dockerhub, no need modify; modify nodename to your edge node hostname, modify configmap name to the result when using *$ kubectl get cm* command

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coap-mapper
spec:
  replicas: 1
  selector:
    matchLabels:
      app: coapmapper
  template:
    metadata:
      labels:
        app: coapmapper
    spec:
      hostNetwork: true
      containers:
      - name: coap-mapper-container
        image: coap-mapper:v1.0.3-linux
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
        volumeMounts:
        - name: config-volume
          mountPath: /opt/kubeedge/
        - mountPath: /dev/ttyS0
          name: coap-dev0
        - mountPath: /dev/ttyS1
          name: coap-dev1
      nodeName: edge120
      volumes:
      - name: config-volume
        configMap:
          name: device-profile-config-edge120
      - name: coap-dev0
        hostPath:
          path: /dev/ttyS0
      - name: coap-dev1
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
    + you will see coap-mapper docker image start automatically
5. I can't find suitable coap server simulator from internet, so I write a coap simulator, it is on project [zhanghongzhi1234/go-coap](https://github.com/zhanghongzhi1234/go-coap), use example\coap_server as simulator, I already build a windows exe, if you need linux version, just upload to a linux server and use go build, it will create binary in linux, it not depend on any third party library, run this simulator and see help, detail enough
  + set [path value] commadn to set path:value
  + set temperature 21
  + set temperature/enable 11
6. Check if data read successfully:
    + execute below command on edge node to view if docker image successfully read the data from terminal:
    + $ docker logs –f \<containerid\> 
    + you will see temperature updated as 26 and temperatue-enable become 11 in the scroll screen in edge node
    + execute below command on master node to view if cloud side successfully get the data, should be realtime update:
    + $ kubectl get devices xxxxx -o yaml –w
    + you will see temperature updated as 26 and temperatue-enable become 11 in the scroll screen in master node too
7. Test write: change coap-device-instance.yaml desired value, then apply again, you will see docker logs, send put request of new desired value to coap server. In real environment, we usually use k8s api to perform write function, and it can be combined with front end dashboard.


## Contributing

PRs accepted.

## License

KubeEdge is under the Apache 2.0 license. See the [LICENSE](../../LICENSE) file for details..