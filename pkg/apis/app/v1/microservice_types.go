/*
Copyright 2019 Hypo.

Licensed under the GNU General Public License, Version 3 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/Coderhypo/KubeService/blob/master/LICENSE

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Canary struct {
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=1
	Weight int `json:"weight"`

	// +optional
	CanaryIngressName string `json:"canaryIngressName,omitempty"`

	// +optional
	Header string `json:"header,omitempty"`

	// +optional
	HeaderValue string `json:"headerValue,omitempty"`

	// +optional
	Cookie string `json:"cookie,omitempty"`
}

type DeployVersion struct {
	Name     string                `json:"name"`
	Template appsv1.DeploymentSpec `json:"template"`

	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// +optional
	Canary *Canary `json:"canary,omitempty"`
}

type ServiceLoadBalance struct {
	Name string             `json:"name"`
	Spec corev1.ServiceSpec `json:"spec"`
}

type IngressLoadBalance struct {
	Name string                        `json:"name"`
	Spec extensionsv1beta1.IngressSpec `json:"spec"`
}

type LoadBalance struct {
	// +optional
	Service *ServiceLoadBalance `json:"service,omitempty"`
	// +optional
	Ingress *IngressLoadBalance `json:"ingress,omitempty"`
}

// MicroServiceSpec defines the desired state of MicroService
type MicroServiceSpec struct {
	// +optional
	LoadBalance        *LoadBalance    `json:"loadBalance,omitempty"`
	Versions           []DeployVersion `json:"versions"`
	CurrentVersionName string          `json:"currentVersionName"`
}

// MicroServiceStatus defines the observed state of MicroService
type MicroServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions        []MicroServiceCondition `json:"conditions,omitempty"`
	AvailableVersions int32                   `json:"availableVersions,omitempty" protobuf:"varint,4,opt,name=availableVersions"`
	TotalVersions     int32                   `json:"totalVersions,omitempty" protobuf:"varint,4,opt,name=totalVersions"`
}

type MicroServiceConditionType string

const (
	MicroServiceAvailable MicroServiceConditionType = "Available"
	MicroServiceProgressing MicroServiceConditionType = "Progressing"
)

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type MicroServiceCondition struct {
	// Type of deployment condition.
	Type MicroServiceConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=MicroServiceConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,6,opt,name=lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,7,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MicroService is the Schema for the microservices API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type MicroService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroServiceSpec   `json:"spec,omitempty"`
	Status MicroServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MicroServiceList contains a list of MicroService
type MicroServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicroService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicroService{}, &MicroServiceList{})
}
