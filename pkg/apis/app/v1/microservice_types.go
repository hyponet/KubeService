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

	CanaryIngressName string `json:"canaryIngressName"`
	Header            string `json:"header"`
	HeaderValue       string `json:"headerValue"`
	Cookie            string `json:"cookie"`
}

type DeployVersion struct {
	Name     string                `json:"name"`
	Template appsv1.DeploymentSpec `json:"template"`
	// +optional
	ServiceName string `json:"serviceName"`
	// +optional
	Canary Canary `json:"canary,omitempty"`
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
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MicroService is the Schema for the microservices API
// +k8s:openapi-gen=true
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
