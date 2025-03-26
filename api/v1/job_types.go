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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.
// Any new fields you add must have json tags for the fields to be serialized.

// JobSpec defines the desired state of Job.
type JobSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Container image to use
	Image string `json:"image,omitempty"`

	// Model to train
	Model string `json:"model,omitempty"`

	// Disk size in GB for the model
	DiskSize int32 `json:"diskSize,omitempty"`

	// Extra arguments to pass to the container
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// HuggingFace secret for downloading the model
	HuggingFaceSecret string `json:"huggingFaceModel,omitempty"`
}

// JobStatus defines the observed state of Job.
type JobStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Job is the Schema for the jobs API.
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec,omitempty"`
	Status JobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobList contains a list of Job.
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
