package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseComponent_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewDatabaseComponent(tctx)
	assert.Equal(t, "database", c.Name())
}

func TestDatabaseComponent_BuildCNPGDatabase(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewDatabaseComponent(tctx)

	db := c.buildCNPGDatabase()

	// Should be in platform namespace, not tenant namespace
	assert.Equal(t, "supabase-system", db.Namespace)
	assert.Equal(t, "main-db-acme", db.Name)

	// Spec fields
	assert.Equal(t, "acme", db.Spec.Name)
	assert.Equal(t, "postgres", db.Spec.Owner)
	assert.Equal(t, "main-db", db.Spec.ClusterRef.Name)

	// Labels
	assert.Equal(t, "acme", db.Labels["supabase.codanael.io/tenant"])
	assert.Equal(t, "supabase-operator", db.Labels["app.kubernetes.io/managed-by"])
}

func TestDatabaseComponent_BuildInitJob(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewDatabaseComponent(tctx)

	job, err := c.buildInitJob()
	require.NoError(t, err)

	// Should be in platform namespace
	assert.Equal(t, "supabase-system", job.Namespace)
	assert.Equal(t, "main-db-init-acme", job.Name)

	// BackoffLimit
	require.NotNil(t, job.Spec.BackoffLimit)
	assert.Equal(t, int32(3), *job.Spec.BackoffLimit)

	// Container
	require.Len(t, job.Spec.Template.Spec.Containers, 1)
	container := job.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/postgres:15.8.1.085", container.Image)

	// Env must include PGPASSWORD
	envMap := envToMap(container.Env)
	assert.Equal(t, "test-db-password", envMap["PGPASSWORD"])

	// Labels
	assert.Equal(t, "acme", job.Labels["supabase.codanael.io/tenant"])
}
