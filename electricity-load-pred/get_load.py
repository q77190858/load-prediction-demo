from kubernetes import  client,config
from kubernetes.stream import stream
import json
config_file = "/home/juju/.kube/config"
config.kube_config.load_kube_config(config_file=config_file)
Api_Instance = client.CoreV1Api()
Api_Batch = client.BatchV1Api()
Api_CO=client.CustomObjectsApi()

api_response = Api_CO.get_namespaced_custom_object(
    group="devices.kubeedge.io",
    version="v1alpha2",
    namespace="default",
    plural="devices",
    name="electricity")
# print(api_response)
print(api_response["status"]["twins"][0]["reported"]["value"])

power=int(api_response["status"]["twins"][0]["reported"]["value"][:-1])
print(power)