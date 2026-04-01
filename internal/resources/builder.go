package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type DeploymentBuilder struct {
	namespace      string
	name           string
	labels         map[string]string
	selectorLabels map[string]string
	podAnnotations map[string]string
	replicas       int32
	containers     []corev1.Container
	volumes        []corev1.Volume
}

func NewDeploymentBuilder(namespace, name string) *DeploymentBuilder {
	return &DeploymentBuilder{
		namespace: namespace,
		name:      name,
		replicas:  1,
	}
}

func (b *DeploymentBuilder) WithLabels(labels map[string]string) *DeploymentBuilder {
	b.labels = labels
	return b
}

func (b *DeploymentBuilder) WithSelectorLabels(labels map[string]string) *DeploymentBuilder {
	b.selectorLabels = labels
	return b
}

func (b *DeploymentBuilder) WithPodAnnotations(annotations map[string]string) *DeploymentBuilder {
	b.podAnnotations = annotations
	return b
}

func (b *DeploymentBuilder) WithReplicas(replicas int32) *DeploymentBuilder {
	b.replicas = replicas
	return b
}

func (b *DeploymentBuilder) WithContainer(c corev1.Container) *DeploymentBuilder {
	b.containers = append(b.containers, c)
	return b
}

func (b *DeploymentBuilder) WithVolumes(vols ...corev1.Volume) *DeploymentBuilder {
	b.volumes = append(b.volumes, vols...)
	return b
}

func (b *DeploymentBuilder) Build() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name,
			Namespace: b.namespace,
			Labels:    b.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &b.replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: b.selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      mergeLabels(b.selectorLabels, b.labels),
					Annotations: b.podAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: b.containers,
					Volumes:    b.volumes,
				},
			},
		},
	}
}

type ServiceBuilder struct {
	namespace      string
	name           string
	labels         map[string]string
	selectorLabels map[string]string
	ports          []corev1.ServicePort
}

func NewServiceBuilder(namespace, name string) *ServiceBuilder {
	return &ServiceBuilder{
		namespace: namespace,
		name:      name,
	}
}

func (b *ServiceBuilder) WithLabels(labels map[string]string) *ServiceBuilder {
	b.labels = labels
	return b
}

func (b *ServiceBuilder) WithSelectorLabels(labels map[string]string) *ServiceBuilder {
	b.selectorLabels = labels
	return b
}

func (b *ServiceBuilder) WithPort(name string, port, targetPort int32) *ServiceBuilder {
	b.ports = append(b.ports, corev1.ServicePort{
		Name:       name,
		Port:       port,
		TargetPort: intstr.FromInt32(targetPort),
		Protocol:   corev1.ProtocolTCP,
	})
	return b
}

func (b *ServiceBuilder) Build() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name,
			Namespace: b.namespace,
			Labels:    b.labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: b.selectorLabels,
			Ports:    b.ports,
		},
	}
}

func mergeLabels(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
