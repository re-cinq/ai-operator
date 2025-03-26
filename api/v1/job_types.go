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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jobDefaultImageName        = "silentehrec/torchtune:latest"
	jobDefaultModelName        = "Qwen/Qwen2.5-0.5B-Instruct"
	jobDefaultDiskSize         = 50
	jobDefaultStorageClassName = "local-path"
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

	// Set the storage class for the disk
	StorageClassName string `json:"storageClassName,omitempty"`

	// Access modes for the disk
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// Command to run in the container
	Command []string `json:"command,omitempty"`

	// HuggingFace secret for downloading the model
	HuggingFaceSecret string `json:"huggingFaceSecret,omitempty"`
}

func (js *JobSpec) Validate() error {
	// Validate the Image field
	if js.Image == "" {
		js.Image = jobDefaultImageName
	}

	// Validate the Model field
	if js.Model == "" {
		js.Model = jobDefaultModelName
	}

	// Validate the DiskSize field
	if js.DiskSize <= 0 {
		js.DiskSize = jobDefaultDiskSize
	}

	// Validate the StorageClassName field
	if js.StorageClassName == "" {
		js.StorageClassName = jobDefaultStorageClassName
	}

	// Validate the AccessModes field
	if len(js.AccessModes) == 0 {
		js.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	// Validate the Command field
	if len(js.Command) == 0 {
		js.Command = []string{
			"tune",
			"run",
			"full_finetune_single_device",
			"-r=3",
			"--config",
			"qwen2_5/0.5B_full_single_device",
		}
	}

	// Validate the HuggingFaceSecret field
	if js.HuggingFaceSecret == "" {
		return fmt.Errorf("HuggingFaceSecret is required")
	}

	return nil
}

// JobStatus defines the observed state of Job.
type JobStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	State   string `json:"state,omitempty"`
	Details string `json:"details,omitempty"`
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
