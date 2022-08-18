# KubeEdge运行load-prediction-demo应用

我们运行load-prediction-demo

由于没有硬件设备，所以本示例中注释了硬件相关的代码，并通过真实统计数据复现电力负载的变化。

## 环境准备

从github下载demo代码

```bash
git clone https://github.com/q77190858/load-prediction-demo
# 可以使用加速
git clone https://gitclone.com/github.com/q77190858/load-prediction-demo
```

进入load-prediction-demo文件夹

```bash
cd ~/load-prediction-demo
```

节点状态为

```bash
kubectl get nodes
NAME        STATUS   ROLES                  AGE     VERSION
edgenode1   Ready    agent,edge             6d17h   v1.22.6-kubeedge-v1.10.0
edgenode2   Ready    agent,edge             6d17h   v1.22.6-kubeedge-v1.10.0
master      Ready    control-plane,master   6d20h   v1.22.11
node        Ready    <none>                 6d19h   v1.22.11
```

## 构建electricity-load-capture的mapper镜像

进入temperature-demo根文件夹

```bash
cd ~/load-prediction-demo
```

构建镜像

```bash
sudo docker build -t electricity-load-capture:test1 .
```

我们使用save+scp+load的方式将打包好的镜像复制到边缘节点

```bash
sudo docker save -o  electricity-load-capture.tar  electricity-load-capture:test-1234
scp  electricity-load-capture.tar  juju@edgenode1:~
docker load -i  electricity-load-capture.tar # 在边缘侧载入镜像
```

## 部署模拟温度设备的model.yaml和instant.yaml

部署model

```bash
cd ~/examples/temperature-demo/crds
kubectl apply -f model.yaml
```

修改instant.yaml

```bash
vim instant.yaml
```

```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: Device
metadata:
  name: electricity
  labels:
    description: 'electricity'
    manufacturer: 'test'
spec:
  deviceModelRef:
    name: electricity-model
  nodeSelector:
    nodeSelectorTerms:
      - matchExpressions:
          - key: ''
            operator: In
            values:
              - edgenode1 # 把这个改成你想部署的边缘节点名称
status:
  twins:
    - propertyName: electricity-status
      desired:
        metadata:
          type: string
        value: ''
```

部署instant.yaml

```bash
kubectl apply -f instant.yaml
```

## 在edge节点部署electricity load capture应用

```bash
cd ~/examples/temperature-demo
vim deployment.yaml
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: electricity-load-capture
  labels:
    app: electricity
spec:
  replicas: 1
  selector:
    matchLabels:
      app: electricity
  template:
    metadata:
      labels:
        app: electricity
    spec:
      hostNetwork: true
      nodeSelector:
        kubernetes.io/hostname: "edgenode1" # 这里改成你的边缘节点的名称
      containers:
      - name: electricity
        image: electricity-load-capture
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
```

部署

```bash
kubectl apply -f  deployment.yaml
```

在master查看pod运行情况

```bash
kubectl get pod
NAME                                 READY   STATUS    RESTARTS      AGE
temperature-mapper-6777c8d86-9gm45   1/1     Running   0             4h31m
```

正常运行

## 在master观察electricity的变化情况

重复执行下面的命令，可以得到随机的功率

```bash
kubectl get device electricity -o yaml
```

```yaml
......
status:
  twins:
  - desired:
      metadata:
        type: string
      value: ""
    propertyName: electricity-status
    reported:
      metadata:
        timestamp: "1656335540900"
        type: string
      value: 94W
```

最后一行就是随机产生的电力负载

## 配置云端推理应用

在主机构建eletricity-load-pred容器

```bash
cd eletricity-load-pred
sudo docker build -t electricity-load-pred:test1 .
sudo docker save -o electricity-load-pred.tar electricity-load-pred:test1
scp electricity-load-pred.tar juju@node:~
```

在node导入eletricity-load-pred容器

```bash
sudo docker load -i eletricity-load-pred.tar
```

在master部署deploy-pred.yaml

```bash
kubectl apply -f deploy-pred.yaml
```

可以看到推理应用已经运行

```bash
kubectl get pod -owide
NAME                                        READY   STATUS    RESTARTS      AGE   IP               NODE        NOMINATED NODE   READINESS GATES
electricity-load-capture-64c57c6b6b-g65z8   1/1     Running   1 (79m ago)   17h   192.168.44.163   edgenode1   <none>           <none>
electricity-load-pred-7b956886dd-t44c4      1/1     Running   0             13s   192.168.44.162   node        <none>           <none>
```

使用logs查看推理输出

```bash
kubectl logs electricity-load-pred-7b956886dd-t44c4
......
prediction
[23.74021624 24.92568992 24.65661687  6.96956846 25.14042489 23.78728854
 24.97256113 24.70614301  6.98240578 25.1900374 ]
```

可以看到输出的预测信息