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

type SupabaseTenantSpec struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]*[a-z0-9]$`
	TenantID    string `json:"tenantId"`
	SupabaseRef string `json:"supabaseRef"`
}

type SupabaseTenantStatus struct {
	Phase              string             `json:"phase,omitempty"`
	// +optional
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	Namespace          string             `json:"namespace,omitempty"`
	Endpoint           string             `json:"endpoint,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenant ID",type=string,JSONPath=`.spec.tenantId`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type SupabaseTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   SupabaseTenantSpec   `json:"spec,omitempty"`
	Status SupabaseTenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SupabaseTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SupabaseTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SupabaseTenant{}, &SupabaseTenantList{})
}
