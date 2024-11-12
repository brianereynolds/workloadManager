# workloadmanager
Workload Manager is used move AKS workloads between different node pools using affinity. 

## Description
Some applications cannot take full advantage of the Kubernetes high-availability concepts. This CRD has been designed to manage these workloads. 

Based on the YAML you provide, the affinity will be adjusted to change the provided key and value. Kubernetes will the re-schedule the workload based on the updated configuration.

It can be thought of as a blue/green deployment pattern, but all within a single AKS cluster (for example 2 node pools, one green, one blue)

## Getting Started

### Prerequisites
- Access to a Azure Kubernetes Service 1.26+
- One or more Deployments or Statefulsets which are scheduled with node affinity 
- A Managed Identity with AKS Contributor Role for the cluster

### Deployment
Helm chart:
```
kubectl create ns operations
helm repo add k8smanagers https://k8smanagers.blob.core.windows.net/helm/
helm install workloadmanager k8smanagers/workloadmanager -n operations
```

#### Verification
```
helm list --filter 'workloadmanager' 
kubectl get crd workloadmanagers.k8smanagers.greyridge.com
```

### Configuration
To activate the CRD configuration, apply a YAML

```yaml
apiVersion: k8smanagers.greyridge.com/v1
kind: WorkloadManager
metadata:
  labels:
    app.kubernetes.io/name: workloadmanager
    app.kubernetes.io/managed-by: kustomize
  name: workloadmanager-sample
spec:
  # Subscription ID of the cluster
  subscriptionId: "3e54eb54-xxxx-yyyy-zzzz-d7b190cd45cf" 
  
  # Resource group containing the cluster
  resourceGroup: "node-upgrader"
  
  # The name of the cluster
  clusterName: "lm-cluster"
  
  # When this flag is true, if an error occurs, the CRD will ask Kubernetes to retry it automatically
  # This will result in the CRD being re-executed with the current configuration.
  # When this flag is false, the CRD will only be executed when the configuration is new/changed.
  retryOnError: false
  
  # When this flag is true, no changes will be executed, only logged
  testMode: false
  
  # A list of procedures (a list of workloads to move)
  procedures:
      # A human-friendly description 
    - description: "move-services"
      # The type of resource to change: deployment|statefulset
      type: "deployment"
      # The namespace that contains the resource
      namespace: "my-app-ns"
      # A list of the resources to change.∑
      workloads:
        - "app1"
        - "app2"
        - "app3"
      # The node affinity to change.
      #    key: The key/label
      #    initial: The existing value of this key (used for verification/rollback)
      #    target: The target value for this key
      affinity:
        key: "agentpool"
        initial: "servicesblue"
        target: "servicesglas"
      # The time, in seconds, to wait for the pods to activate after changing the node affinity.
      # If the pod doesn't activate within this time period, it will move to the next 
      # workload in the list (the pod might well activate in time). If not provided, defaults to 600
      timeout: 600

```

## Details

### Affinity

The CRD assumes you have node affinity for _statefulset_ and/or _deployment_ like this (key and value are user-defined)

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: agentpool
              operator: In
              values:
                - servicesblue
```

### Uninstall
Uninstall the helm chart
```
helm uninstall -n operations workloadlmanager
```

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

