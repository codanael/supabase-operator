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

// --- Tenant-specific types ---

// TenantAuthSpec configures the GoTrue auth service for a tenant.
type TenantAuthSpec struct {
	// +optional
	SiteURL string `json:"siteURL,omitempty"`

	// +optional
	AdditionalRedirectURLs []string `json:"additionalRedirectURLs,omitempty"`

	// +optional
	DisableSignup bool `json:"disableSignup,omitempty"`

	// +optional
	Email *EmailAuthSpec `json:"email,omitempty"`

	// +optional
	SMTP *SMTPSpec `json:"smtp,omitempty"`

	// +optional
	External *OAuthProviders `json:"external,omitempty"`
}

// EmailAuthSpec configures email-based authentication.
type EmailAuthSpec struct {
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`

	// +optional
	Autoconfirm bool `json:"autoconfirm,omitempty"`
}

// SMTPSpec configures SMTP for sending auth emails.
type SMTPSpec struct {
	Host string `json:"host"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// CredentialsSecret is the name of a Secret containing "username" and "password" keys.
	CredentialsSecret string `json:"credentialsSecret"`

	// +optional
	SenderName string `json:"senderName,omitempty"`
}

// OAuthProviders configures external OAuth providers.
type OAuthProviders struct {
	// +optional
	Google *OAuthProvider `json:"google,omitempty"`
	// +optional
	GitHub *OAuthProvider `json:"github,omitempty"`
	// +optional
	Azure *OAuthProvider `json:"azure,omitempty"`
}

// OAuthProvider configures a single OAuth provider.
type OAuthProvider struct {
	Enabled bool `json:"enabled"`

	// CredentialsSecret is the name of a Secret containing "clientId" and "clientSecret" keys.
	CredentialsSecret string `json:"credentialsSecret"`
}

// TenantRESTSpec configures the PostgREST API service.
type TenantRESTSpec struct {
	// +kubebuilder:default:={"public","graphql_public"}
	// +optional
	Schemas []string `json:"schemas,omitempty"`

	// +kubebuilder:default:=1000
	// +optional
	MaxRows *int32 `json:"maxRows,omitempty"`
}

// TenantRealtimeSpec configures the Realtime service.
type TenantRealtimeSpec struct {
	ServiceSpec `json:",inline"`
}

// StorageBackend defines the backend type for tenant storage.
// +kubebuilder:validation:Enum=file;s3;obc
type StorageBackend string

const (
	StorageBackendFile StorageBackend = "file"
	StorageBackendS3   StorageBackend = "s3"
	StorageBackendOBC  StorageBackend = "obc"
)

// TenantStorageSpec configures the Storage service.
type TenantStorageSpec struct {
	// +kubebuilder:default:="file"
	// +optional
	Backend StorageBackend `json:"backend,omitempty"`

	// +kubebuilder:default:=52428800
	// +optional
	FileSizeLimit *int64 `json:"fileSizeLimit,omitempty"`

	// +optional
	ImageTransformation bool `json:"imageTransformation,omitempty"`

	// +optional
	S3 *S3Config `json:"s3,omitempty"`

	// +optional
	ObjectBucket *ObjectBucketSpec `json:"objectBucket,omitempty"`
}

// S3Config configures S3-compatible storage.
type S3Config struct {
	Bucket string `json:"bucket"`
	Region string `json:"region"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`

	// CredentialsSecret is the name of a Secret containing "accessKeyId" and "secretAccessKey" keys.
	CredentialsSecret string `json:"credentialsSecret"`
}

// ObjectBucketSpec configures OBC-based storage using the ObjectBucketClaim API.
type ObjectBucketSpec struct {
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// +optional
	AdditionalConfig map[string]string `json:"additionalConfig,omitempty"`

	// +optional
	BucketPrefix string `json:"bucketPrefix,omitempty"`
}

// TenantFunctionsSpec configures the Edge Functions runtime.
type TenantFunctionsSpec struct {
	// +kubebuilder:default:=true
	// +optional
	VerifyJWT bool `json:"verifyJWT,omitempty"`

	// +optional
	Source *FunctionSource `json:"source,omitempty"`
}

// FunctionSource defines where edge function code is loaded from.
type FunctionSource struct {
	// ConfigMapRef is the name of a ConfigMap containing function source code.
	ConfigMapRef string `json:"configMapRef"`
}

// ResourcePreset defines a resource sizing preset.
// +kubebuilder:validation:Enum=small;medium;large;custom
type ResourcePreset string

const (
	ResourcePresetSmall  ResourcePreset = "small"
	ResourcePresetMedium ResourcePreset = "medium"
	ResourcePresetLarge  ResourcePreset = "large"
	ResourcePresetCustom ResourcePreset = "custom"
)
