package resources

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
)

// PresetResources defines the resource sizing for a given preset.
type PresetResources struct {
	Replicas  int32
	Resources corev1.ResourceRequirements
}

// GetPreset returns the resource sizing for the given preset.
func GetPreset(preset v1alpha1.ResourcePreset) PresetResources {
	switch preset {
	case v1alpha1.ResourcePresetSmall:
		return PresetResources{
			Replicas: 1,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		}
	case v1alpha1.ResourcePresetMedium:
		return PresetResources{
			Replicas: 2,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1000m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
	case v1alpha1.ResourcePresetLarge:
		return PresetResources{
			Replicas: 3,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2000m"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
		}
	default: // custom - no preset, use defaults
		return PresetResources{
			Replicas:  1,
			Resources: corev1.ResourceRequirements{},
		}
	}
}
