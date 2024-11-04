/*
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
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkloadTypes string

const (
	StatefulSet = "statefulset"
	Deployment  = "deployment"
)

type Affinity struct {
	Key     string `json:"key,omitempty"`
	Initial string `json:"initial,omitempty"`
	Target  string `json:"target,omitempty"`
}

type Procedure struct {
	Description string        `json:"description,omitempty"`
	Type        WorkloadTypes `json:"type,omitempty"`
	Namespace   string        `json:"namespace,omitempty"`
	Workloads   []string      `json:"workloads"`
	Affinity    Affinity      `json:"affinity"`
	Timeout     int           `json:"timeout,omitempty"`
}

// WorkloadManagerSpec defines the desired state of WorkloadManager
type WorkloadManagerSpec struct {
	SubscriptionID string      `json:"subscriptionId,omitempty"`
	ResourceGroup  string      `json:"resourceGroup,omitempty"`
	ClusterName    string      `json:"clusterName,omitempty"`
	RetryOnError   bool        `json:"retryOnError,omitempty"`
	TestMode       bool        `json:"testMode,omitempty"`
	Procedures     []Procedure `json:"procedures"`
}

// WorkloadManagerStatus defines the observed state of WorkloadManager
type WorkloadManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WorkloadManager is the Schema for the workloadmanagers API
type WorkloadManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadManagerSpec   `json:"spec,omitempty"`
	Status WorkloadManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadManagerList contains a list of WorkloadManager
type WorkloadManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadManager{}, &WorkloadManagerList{})
}
