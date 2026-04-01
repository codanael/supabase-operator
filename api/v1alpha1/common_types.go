/*
Copyright 2026.

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
)

// DatabaseSpec configures the shared CNPG PostgreSQL cluster.
type DatabaseSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=3
	Instances int32 `json:"instances"`

	// +kubebuilder:default:="supabase/postgres:15.8.1.085"
	ImageName string `json:"imageName,omitempty"`

	Storage PersistentStorageSpec `json:"storage"`

	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type PersistentStorageSpec struct {
	// +kubebuilder:default:="10Gi"
	Size string `json:"size"`

	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

type BackupSpec struct {
	Schedule        string             `json:"schedule"`
	DestinationPath string             `json:"destinationPath"`
	S3Credentials   *S3CredentialsSpec `json:"s3Credentials,omitempty"`
}

type S3CredentialsSpec struct {
	SecretRef string `json:"secretRef"`
}

type GatewaySpec struct {
	GatewayClassName string `json:"gatewayClassName"`
	BaseDomain       string `json:"baseDomain"`
	// +optional
	TLS *GatewayTLSSpec `json:"tls,omitempty"`
}

type GatewayTLSSpec struct {
	CertificateSecretRef string `json:"certificateSecretRef"`
}

type ImageOverrides struct {
	// +optional
	Imgproxy *string `json:"imgproxy,omitempty"`
	// +optional
	Studio *string `json:"studio,omitempty"`
	// +optional
	Analytics *string `json:"analytics,omitempty"`
	// +optional
	Vector *string `json:"vector,omitempty"`
	// +optional
	Supavisor *string `json:"supavisor,omitempty"`
}

type ServiceSpec struct {
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

func (s *ServiceSpec) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

func (s *ServiceSpec) GetReplicas() int32 {
	if s.Replicas == nil {
		return 1
	}
	return *s.Replicas
}

type ImgproxySpec struct {
	ServiceSpec `json:",inline"`
}

type StudioSpec struct {
	ServiceSpec `json:",inline"`
}

type AnalyticsSpec struct {
	ServiceSpec `json:",inline"`
}

type VectorSpec struct {
	ServiceSpec `json:",inline"`
}

type SupavisorSpec struct {
	ServiceSpec `json:",inline"`
}

type ComponentStatus struct {
	Ready bool `json:"ready"`
	// +optional
	Message string `json:"message,omitempty"`
	// +optional
	Version string `json:"version,omitempty"`
}
