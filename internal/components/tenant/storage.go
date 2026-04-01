package tenant

import (
	"fmt"
	"strconv"

	"github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentStorage    = "storage"
	defaultStorageImage = "supabase/storage-api:v1.44.2"
	storagePort         = 5000
)

// StorageBuilder builds the Storage API deployment and service.
type StorageBuilder struct {
	ctx *components.TenantContext
}

func NewStorageBuilder(ctx *components.TenantContext) *StorageBuilder {
	return &StorageBuilder{ctx: ctx}
}

func (b *StorageBuilder) ComponentName() string {
	return componentStorage
}

func (b *StorageBuilder) resourceName() string {
	return fmt.Sprintf("%s-storage", b.ctx.TenantID())
}

func (b *StorageBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentStorage)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentStorage)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	storageSpec := b.ctx.Tenant.Spec.Storage

	fileSizeLimit := "52428800"
	if storageSpec.FileSizeLimit != nil {
		fileSizeLimit = strconv.FormatInt(*storageSpec.FileSizeLimit, 10)
	}

	env := []corev1.EnvVar{
		{Name: "ANON_KEY", Value: b.ctx.AnonKey},
		{Name: "SERVICE_KEY", Value: b.ctx.ServiceRoleKey},
		{Name: "AUTH_JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "DATABASE_URL", Value: b.ctx.DatabaseDSN("supabase_storage_admin", b.ctx.DatabasePassword)},
		{Name: "FILE_SIZE_LIMIT", Value: fileSizeLimit},
		{Name: "TENANT_ID", Value: b.ctx.TenantID()},
		{Name: "REGION", Value: "local"},
		{Name: "ENABLE_IMAGE_TRANSFORMATION", Value: strconv.FormatBool(storageSpec.ImageTransformation)},
	}

	// Backend-specific env
	switch storageSpec.Backend {
	case v1alpha1.StorageBackendS3:
		env = append(env, corev1.EnvVar{Name: "STORAGE_BACKEND", Value: "s3"})
		if storageSpec.S3 != nil {
			env = append(env,
				corev1.EnvVar{Name: "GLOBAL_S3_BUCKET", Value: storageSpec.S3.Bucket},
				corev1.EnvVar{Name: "GLOBAL_S3_REGION", Value: storageSpec.S3.Region},
			)
			if storageSpec.S3.Endpoint != "" {
				env = append(env, corev1.EnvVar{Name: "GLOBAL_S3_ENDPOINT", Value: storageSpec.S3.Endpoint})
			}
			if storageSpec.S3.ForcePathStyle {
				env = append(env, corev1.EnvVar{Name: "GLOBAL_S3_FORCE_PATH_STYLE", Value: "true"})
			}
		}
	case v1alpha1.StorageBackendOBC:
		env = append(env, corev1.EnvVar{Name: "STORAGE_BACKEND", Value: "s3"})
	default: // file
		env = append(env,
			corev1.EnvVar{Name: "STORAGE_BACKEND", Value: "file"},
			corev1.EnvVar{Name: "FILE_STORAGE_BACKEND_PATH", Value: "/var/lib/storage"},
		)
	}

	return resources.NewDeploymentBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithContainer(corev1.Container{
			Name:  componentStorage,
			Image: defaultStorageImage,
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: storagePort, Protocol: corev1.ProtocolTCP},
			},
			Env: env,
		}).
		Build()
}

func (b *StorageBuilder) BuildService() *corev1.Service {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentStorage)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentStorage)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	return resources.NewServiceBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", storagePort, storagePort).
		Build()
}

func NewStorage(ctx *components.TenantContext) *TenantServiceComponent {
	return NewTenantServiceComponent(ctx, NewStorageBuilder(ctx))
}
