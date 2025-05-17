package v1alpha1

import corev1 "k8s.io/api/core/v1"

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
