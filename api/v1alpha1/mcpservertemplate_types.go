/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MCPServerTemplateSpec defines the desired state of MCPServerTemplate
type MCPServerTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PodSpec `json:",inline"`
}

// +k8s:openapi-gen=true
type PodSpec struct {
	// List of containers belonging to the pod.
	// Containers cannot currently be added or removed.
	// There must be at least one container in a Pod.
	// Cannot be updated.
	// +patchMergeKey=name
	// +patchStrategy=merge
	Containers []corev1.Container `json:"containers" patchStrategy:"merge" patchMergeKey:"name" validate:"required"`

	// Labels that will be add to the pod.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations that will be add to the pod.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MCPServerTemplateStatus defines the observed state of MCPServerTemplate
type MCPServerTemplateStatus struct {

	// Conditions store the status conditions of the Memcached instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MCPServerTemplate is the Schema for the mcpservertemplates API
type MCPServerTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServerTemplateSpec   `json:"spec,omitempty"`
	Status MCPServerTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MCPServerTemplateList contains a list of MCPServerTemplate
type MCPServerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServerTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MCPServerTemplate{}, &MCPServerTemplateList{})
}
