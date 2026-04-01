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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- Phase constants ---

const (
	TenantPhasePending      = "Pending"
	TenantPhaseProvisioning = "Provisioning"
	TenantPhaseReady        = "Ready"
	TenantPhaseDegraded     = "Degraded"
	TenantPhaseError        = "Error"
	TenantPhaseSuspending   = "Suspending"
	TenantPhaseSuspended    = "Suspended"
	TenantPhaseResuming     = "Resuming"
	TenantPhaseDeleting     = "Deleting"
)

// --- Condition type constants ---

const (
	TenantConditionNamespaceReady = "NamespaceReady"
	TenantConditionSecretsReady   = "SecretsReady"
	TenantConditionDatabaseReady  = "DatabaseReady"
	TenantConditionAuthReady      = "AuthReady"
	TenantConditionRESTReady      = "RESTReady"
	TenantConditionRealtimeReady  = "RealtimeReady"
	TenantConditionStorageReady   = "StorageReady"
	TenantConditionFunctionsReady = "FunctionsReady"
	TenantConditionRoutingReady   = "RoutingReady"
)

// SupabaseTenantSpec defines the desired state of a SupabaseTenant.
type SupabaseTenantSpec struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]*[a-z0-9]$`
	TenantID string `json:"tenantId"`

	// SupabaseRef is the name of the parent Supabase resource (in the same namespace).
	SupabaseRef string `json:"supabaseRef"`

	// +optional
	Auth TenantAuthSpec `json:"auth,omitempty"`

	// +optional
	REST TenantRESTSpec `json:"rest,omitempty"`

	// +optional
	Realtime TenantRealtimeSpec `json:"realtime,omitempty"`

	// +optional
	Storage TenantStorageSpec `json:"storage,omitempty"`

	// +optional
	Functions TenantFunctionsSpec `json:"functions,omitempty"`

	// Suspended, when true, scales tenant workloads to zero.
	// +optional
	Suspended bool `json:"suspended,omitempty"`

	// Resources selects a resource sizing preset for tenant workloads.
	// +kubebuilder:default:="small"
	// +optional
	Resources ResourcePreset `json:"resources,omitempty"`
}

// TenantComponentStatuses tracks readiness of each tenant component.
type TenantComponentStatuses struct {
	Database ComponentStatus `json:"database,omitempty"`
	Auth     ComponentStatus `json:"auth,omitempty"`
	REST     ComponentStatus `json:"rest,omitempty"`
	Realtime ComponentStatus `json:"realtime,omitempty"`
	Storage  ComponentStatus `json:"storage,omitempty"`
	// +optional
	Functions ComponentStatus `json:"functions,omitempty"`
	// +optional
	Routing ComponentStatus `json:"routing,omitempty"`
}

// SupabaseTenantStatus defines the observed state of SupabaseTenant.
type SupabaseTenantStatus struct {
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	Components TenantComponentStatuses `json:"components,omitempty"`

	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenant ID",type=string,JSONPath=`.spec.tenantId`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SupabaseTenant is the Schema for the supabasetenants API.
type SupabaseTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SupabaseTenantSpec   `json:"spec,omitempty"`
	Status            SupabaseTenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SupabaseTenantList contains a list of SupabaseTenant.
type SupabaseTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SupabaseTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SupabaseTenant{}, &SupabaseTenantList{})
}
