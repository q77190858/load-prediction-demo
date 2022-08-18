import numpy as np
from kubernetes import  client,config
from kubernetes.stream import stream
import json
import time
import torch
from torch import nn
from torch.autograd import Variable

torch.set_default_tensor_type(torch.DoubleTensor)

config_file = "config"
config.kube_config.load_kube_config(config_file=config_file)
Api_Instance = client.CoreV1Api()
Api_Batch = client.BatchV1Api()
Api_CO=client.CustomObjectsApi()

dataset=[]
class lstm(nn.Module):
    def __init__(self,input_size=480,hidden_size=480,output_size=480,num_layer=2):
        super(lstm,self).__init__()
        self.layer1 = nn.LSTM(input_size,hidden_size,num_layer)
        self.layer2 = nn.Linear(hidden_size,output_size)

    def forward(self,x):
        x,_ = self.layer1(x)
        s,b,h = x.size()
        x = x.view(s*b,h)
        x = self.layer2(x)
        x = x.view(s,b,-1)
        return x

while True:
    time.sleep(1)

    api_response = Api_CO.get_namespaced_custom_object(
        group="devices.kubeedge.io",
        version="v1alpha2",
        namespace="default",
        plural="devices",
        name="electricity")
    print(api_response["status"]["twins"][0]["reported"]["value"])
    power=int(api_response["status"]["twins"][0]["reported"]["value"][:-1])

    dataset.append(power)
    if len(dataset)<20:
        print("collect enough data!")
        continue
    if len(dataset)>20:
        dataset=dataset[-20:]


    # 划分训练集和测试集，50% 作为训练集
    train_size = int(len(dataset) * 0.5)
    test_size = len(dataset) - train_size
    train_set = dataset[:train_size]
    test_set = dataset[train_size:]
    train_X = np.array(train_set[:train_size//2]).astype(float)
    train_Y = np.array(train_set[train_size//2:]).astype(float)
    test_X = np.array(test_set[test_size//2:]).astype(float)
    test_Y = np.array(test_set[:test_size//2]).astype(float)

    # 设置LSTM能识别的数据类型，形成tran_X的一维两个参数的数组，train_Y的一维一个参数的数组。并转化为tensor类型

    # 把list转numpy三维数组，第一维自适应
    train_X = train_X.reshape(-1, 1, 5)
    train_Y = train_Y.reshape(-1, 1, 5)
    test_X = test_X.reshape(-1, 1, 5)
    # numpy数组转tensor
    train_x = torch.from_numpy(train_X)
    train_y = torch.from_numpy(train_Y)
    test_x = torch.from_numpy(test_X)

    # 建立LSTM模型，第一层为LSTM神经网络，第二层为一个全连接层。
    model = lstm(5, 5,5,2)

    # 设置交叉熵损失函数和自适应梯度下降算法
    criterion = nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=1e-2)

    # 开始训练
    for e in range(500):
        var_x = Variable(train_x)
        var_y = Variable(train_y)
        # 前向传播
        out = model(var_x)
        loss = criterion(out, var_y)
        # 反向传播
        optimizer.zero_grad()
        loss.backward()
        optimizer.step()

        if (e + 1) % 100 == 0: # 每 100 次输出结果
            print('Epoch: {}, Loss: {:.5f}'.format(e + 1, loss.item()))
        
    model = model.eval() # 转换成测试模式

    var_test_x = Variable(test_x)
    pred_test_y = model(var_test_x) # 测试集的预测结果
    # 改变输出的格式
    # pred_test = pred_test.view(-1).data.numpy()
    pred_test_Y = pred_test_y.view(-1).data.numpy()

    # 取预测的结果和实际对比，打印测试集中实际结果和预测的结果
    print("pred_test_Y")
    print(pred_test_Y)
    print("test_Y")
    print(test_Y)


    # 分析一下误差
    # 均方误差
    MSE = np.linalg.norm(test_Y-pred_test_Y, ord=2)**2/len(test_Y)
    # 平均绝对误差
    MAE = np.linalg.norm(test_Y-pred_test_Y, ord=1)/len(test_Y)
    # 平均绝对百分比误差
    MAPE = np.mean(np.abs((test_Y-pred_test_Y) / test_Y)) * 100
    # 模型的准确率
    Accuracy=(1-np.sqrt(np.mean(((test_Y-pred_test_Y) / test_Y)**2)))*100

    print("MSE={}\nMAE={}\nMAPE={}%\nAccuracy={}%\n".format(MSE,MAE,MAPE,Accuracy))

    # 使用滑动窗口预测未来的10次
    pred_list=[]
    x=dataset[-5:]
    for i in range(2):
        input_x=[]
        input_x.append(x)
        input_x=np.array(input_x).astype(float)
        input_x=input_x.reshape(-1,1,5)
        var_input_x=Variable(torch.from_numpy(input_x))
        pred=model(var_input_x)
        pred=pred.view(-1).data.numpy()
        x=pred
        pred_list.append(pred)

    prediction=np.array(pred_list).reshape(-1)
    print("prediction")
    print(prediction)