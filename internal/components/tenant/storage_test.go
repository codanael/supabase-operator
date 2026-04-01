package tenant

import (
	"testing"

	"github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewStorage(tctx)
	assert.Equal(t, "storage", c.Name())
}

func TestStorage_BuildDeployment(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewStorageBuilder(tctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "acme-storage", deploy.Name)
	assert.Equal(t, "supabase-acme", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/storage-api:v1.44.2", c.Image)
	assert.Equal(t, int32(5000), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "test-anon-key", envMap["ANON_KEY"])
	assert.Equal(t, "test-service-role-key", envMap["SERVICE_KEY"])
	assert.Equal(t, "test-jwt-secret-base64encoded", envMap["AUTH_JWT_SECRET"])
	assert.Contains(t, envMap["DATABASE_URL"], "supabase_storage_admin")
	assert.Equal(t, "acme", envMap["TENANT_ID"])
	assert.Equal(t, "local", envMap["REGION"])
	// Default backend is file
	assert.Equal(t, "file", envMap["STORAGE_BACKEND"])
	assert.Equal(t, "/var/lib/storage", envMap["FILE_STORAGE_BACKEND_PATH"])
}

func TestStorage_BuildDeployment_S3Backend(t *testing.T) {
	tctx := newTestTenantContext()
	tctx.Tenant.Spec.Storage = v1alpha1.TenantStorageSpec{
		Backend: v1alpha1.StorageBackendS3,
		S3: &v1alpha1.S3Config{
			Bucket:            "my-bucket",
			Region:            "us-east-1",
			Endpoint:          "https://s3.example.com",
			ForcePathStyle:    true,
			CredentialsSecret: "s3-creds",
		},
	}
	b := NewStorageBuilder(tctx)

	deploy := b.BuildDeployment()
	c := deploy.Spec.Template.Spec.Containers[0]
	envMap := envToMap(c.Env)

	assert.Equal(t, "s3", envMap["STORAGE_BACKEND"])
	assert.Equal(t, "my-bucket", envMap["GLOBAL_S3_BUCKET"])
	assert.Equal(t, "us-east-1", envMap["GLOBAL_S3_REGION"])
	assert.Equal(t, "https://s3.example.com", envMap["GLOBAL_S3_ENDPOINT"])
	assert.Equal(t, "true", envMap["GLOBAL_S3_FORCE_PATH_STYLE"])
}

func TestStorage_BuildService(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewStorageBuilder(tctx)

	svc := b.BuildService()

	assert.Equal(t, "acme-storage", svc.Name)
	assert.Equal(t, "supabase-acme", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(5000), svc.Spec.Ports[0].Port)
}
