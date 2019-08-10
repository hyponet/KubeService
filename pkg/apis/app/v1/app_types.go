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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type MicroServiceTemplate struct {
	Name string           `json:"name"`
	Spec MicroServiceSpec `json:"spec,omitempty"`
}

// AppSpec defines the desired state of App
type AppSpec struct {
	MicroServices []MicroServiceTemplate `json:"microServices,omitempty"`
}

type FromManagerType string

const (
	ManagerCreated FromManagerType = "Created"
	ManagerUpdated FromManagerType = "Updated"
	ManagerNone    FromManagerType = "Nil"
)

// AppStatus defines the observed state of App
type AppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions             []AppCondition  `json:"conditions,omitempty"`
	AvailableMicroServices int32           `json:"availableVersions,omitempty" protobuf:"varint,4,opt,name=availableMSs"`
	TotalMicroServices     int32           `json:"totalVersions,omitempty" protobuf:"varint,4,opt,name=totalMSs"`
	FromManager            FromManagerType `json:"fromManager,omitempty" protobuf:"bytes,1,opt,name=type,casttype=FromManagerType"`
}

type AppConditionType string

const (
	AppAvailable   AppConditionType = "Available"
	AppProgressing AppConditionType = "Progressing"
)

type AppCondition struct {
	// Type of deployment condition.
	Type AppConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=AppConditionType"`
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

// App is the Schema for the apps API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
