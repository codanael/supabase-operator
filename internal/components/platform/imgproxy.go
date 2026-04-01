package platform

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentImgproxy    = "imgproxy"
	defaultImgproxyImage = "darthsim/imgproxy:v3.30.1"
	imgproxyPort         = 5001
)

type ImgproxyBuilder struct {
	ctx *components.PlatformContext
}

func NewImgproxyBuilder(ctx *components.PlatformContext) *ImgproxyBuilder {
	return &ImgproxyBuilder{ctx: ctx}
}

func (b *ImgproxyBuilder) ComponentName() string {
	return componentImgproxy
}

func (b *ImgproxyBuilder) IsEnabled() bool {
	return b.ctx.Supabase.Spec.Imgproxy.IsEnabled()
}

func (b *ImgproxyBuilder) resourceName() string {
	return fmt.Sprintf("%s-imgproxy", b.ctx.InstanceName())
}

func (b *ImgproxyBuilder) image() string {
	if b.ctx.Supabase.Spec.Images.Imgproxy != nil {
		return *b.ctx.Supabase.Spec.Images.Imgproxy
	}
	return defaultImgproxyImage
}

func (b *ImgproxyBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentImgproxy)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentImgproxy)

	return resources.NewDeploymentBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(b.ctx.Supabase.Spec.Imgproxy.GetReplicas()).
		WithContainer(corev1.Container{
			Name:  componentImgproxy,
			Image: b.image(),
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: imgproxyPort, Protocol: corev1.ProtocolTCP},
			},
			Env: []corev1.EnvVar{
				{Name: "IMGPROXY_BIND", Value: fmt.Sprintf(":%d", imgproxyPort)},
				{Name: "IMGPROXY_LOCAL_FILESYSTEM_ROOT", Value: "/"},
				{Name: "IMGPROXY_USE_ETAG", Value: "true"},
				{Name: "IMGPROXY_ENABLE_WEBP_DETECTION", Value: "true"},
			},
			Resources: b.ctx.Supabase.Spec.Imgproxy.Resources,
		}).
		Build()
}

func (b *ImgproxyBuilder) BuildService() *corev1.Service {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentImgproxy)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentImgproxy)

	return resources.NewServiceBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", imgproxyPort, imgproxyPort).
		Build()
}

func NewImgproxy(ctx *components.PlatformContext) *ServiceComponent {
	return NewServiceComponent(ctx, NewImgproxyBuilder(ctx))
}
