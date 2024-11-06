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
Helm chart is recommended:
```
cd charts/workloadmanager
helm install workloadmanager .
```

#### Verification
```
helm list --filter 'workloadmanager' 
kubectl get crd workloadmanagers.k8smanageers.greyridge.com
```

### Configuration
To activate the CRD configuration, apply a YAML

```yaml
apiVersion: k8smanageers.greyridge.com/v1
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

```


### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/workloadmanager:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/workloadmanager:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/workloadmanager:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/workloadmanager/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

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

