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

type SupabaseSpec struct {
	Database  DatabaseSpec  `json:"database"`
	Gateway   GatewaySpec   `json:"gateway"`
	// +optional
	Imgproxy  ImgproxySpec  `json:"imgproxy,omitempty"`
	// +optional
	Studio    StudioSpec    `json:"studio,omitempty"`
	// +optional
	Analytics AnalyticsSpec `json:"analytics,omitempty"`
	// +optional
	Vector    VectorSpec    `json:"vector,omitempty"`
	// +optional
	Supavisor SupavisorSpec `json:"supavisor,omitempty"`
	// +optional
	Images    ImageOverrides `json:"images,omitempty"`
}

type SupabaseStatus struct {
	Phase              string             `json:"phase,omitempty"`
	// +optional
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	DatabaseReady      bool               `json:"databaseReady,omitempty"`
	GatewayReady       bool               `json:"gatewayReady,omitempty"`
	TenantCount        int32              `json:"tenantCount,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

const (
	SupphasePending      = "Pending"
	SupphaseProvisioning = "Provisioning"
	SupphaseReady        = "Ready"
	SupphaseDegraded     = "Degraded"
	SupphaseError        = "Error"
	SupphaseUpgrading    = "Upgrading"
)

const (
	ConditionDatabaseReady  = "DatabaseReady"
	ConditionGatewayReady   = "GatewayReady"
	ConditionImgproxyReady  = "ImgproxyReady"
	ConditionStudioReady    = "StudioReady"
	ConditionAnalyticsReady = "AnalyticsReady"
	ConditionVectorReady    = "VectorReady"
	ConditionSupavisorReady = "SupavisorReady"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="DB Ready",type=boolean,JSONPath=`.status.databaseReady`
// +kubebuilder:printcolumn:name="GW Ready",type=boolean,JSONPath=`.status.gatewayReady`
// +kubebuilder:printcolumn:name="Tenants",type=integer,JSONPath=`.status.tenantCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type Supabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   SupabaseSpec   `json:"spec,omitempty"`
	Status SupabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SupabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Supabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Supabase{}, &SupabaseList{})
}
