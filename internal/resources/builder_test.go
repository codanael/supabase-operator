package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildDeployment(t *testing.T) {
	dep := NewDeploymentBuilder("test-ns", "test-deploy").
		WithLabels(PlatformLabels("main", "imgproxy")).
		WithSelectorLabels(SelectorLabels("main", "imgproxy")).
		WithReplicas(2).
		WithContainer(corev1.Container{
			Name:  "imgproxy",
			Image: "darthsim/imgproxy:v3.30.1",
			Ports: []corev1.ContainerPort{{ContainerPort: 5001}},
		}).
		Build()

	assert.Equal(t, "test-ns", dep.Namespace)
	assert.Equal(t, "test-deploy", dep.Name)
	assert.Equal(t, int32(2), *dep.Spec.Replicas)
	require.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "darthsim/imgproxy:v3.30.1", dep.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "imgproxy", dep.Spec.Template.Labels[LabelComponent])
}

func TestBuildService(t *testing.T) {
	svc := NewServiceBuilder("test-ns", "test-svc").
		WithLabels(PlatformLabels("main", "imgproxy")).
		WithSelectorLabels(SelectorLabels("main", "imgproxy")).
		WithPort("http", 5001, 5001).
		Build()

	assert.Equal(t, "test-ns", svc.Namespace)
	assert.Equal(t, "test-svc", svc.Name)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(5001), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt32(5001), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
}
