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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FunctionSpec defines the desired state of Function
type FunctionSpec struct {

	RuntimeImage string `json:"runtimeImage"`

	Code string `json:"code"`

	// FunctionName is the name of the function to be called
	Handler string `json:"handler"`

	// Arguments is a list of arguments to be passed to the function
	Args []string `json:"args,omitempty"`

	// EnvironmentVariables is a map of environment variables to be set
	EnvVariables map[string]string `json:"envVariables,omitempty"`

	// Dependencies is a list of Python packages to be installed
	Dependencies []string `json:"dependencies,omitempty"`

	// Number of pods
	Replicas *int32 `json:"replicas"`

	// TTL after which completed pods will be cleaned up
	// +kubebuilder:default=300
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// FunctionStatus defines the observed state of Function
type FunctionStatus struct {
	Replicas    int32              `json:"replicas"`
	Active 		int32              `json:"active"`
	Completed 	int32              `json:"completed"`
	Created 	int32              `json:"created"`
	Selector    string             `json:"selector,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector

// Function is the Schema for the functions API
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FunctionList contains a list of Function
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Function `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Function{}, &FunctionList{})
}