package platform

import (
	"testing"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNPGCluster_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewCNPGCluster(pctx)
	assert.Equal(t, "database", c.Name())
}

func TestCNPGCluster_BuildCluster(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewCNPGCluster(pctx)

	cluster := c.buildCluster()

	assert.Equal(t, "main-db", cluster.Name)
	assert.Equal(t, "supabase-system", cluster.Namespace)
	assert.Equal(t, 3, cluster.Spec.Instances)
	assert.Equal(t, "supabase/postgres:15.8.1.085", cluster.Spec.ImageName)
	assert.Equal(t, "50Gi", cluster.Spec.StorageConfiguration.Size)
	assert.Nil(t, cluster.Spec.Backup)

	// Check labels
	assert.Equal(t, "supabase-operator", cluster.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "supabase", cluster.Labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "main", cluster.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, "database", cluster.Labels["app.kubernetes.io/component"])
}

func TestCNPGCluster_BuildCluster_WithBackup(t *testing.T) {
	sb := newTestSupabase()
	sb.Spec.Database.Backup = &v1alpha1.BackupSpec{
		Schedule:        "0 0 * * *",
		DestinationPath: "s3://my-bucket/backups",
		S3Credentials: &v1alpha1.S3CredentialsSpec{
			SecretRef: "backup-creds",
		},
	}
	pctx := newTestPlatformContext(sb)
	c := NewCNPGCluster(pctx)

	cluster := c.buildCluster()

	require.NotNil(t, cluster.Spec.Backup)
	require.NotNil(t, cluster.Spec.Backup.BarmanObjectStore)
	assert.Equal(t, "s3://my-bucket/backups", cluster.Spec.Backup.BarmanObjectStore.DestinationPath)

	require.NotNil(t, cluster.Spec.Backup.BarmanObjectStore.AWS)
	assert.Equal(t, "backup-creds", cluster.Spec.Backup.BarmanObjectStore.AWS.AccessKeyIDReference.Name)
	assert.Equal(t, "ACCESS_KEY_ID", cluster.Spec.Backup.BarmanObjectStore.AWS.AccessKeyIDReference.Key)
	assert.Equal(t, "backup-creds", cluster.Spec.Backup.BarmanObjectStore.AWS.SecretAccessKeyReference.Name)
	assert.Equal(t, "ACCESS_SECRET_KEY", cluster.Spec.Backup.BarmanObjectStore.AWS.SecretAccessKeyReference.Key)
}

func TestCNPGCluster_BuildCluster_WithStorageClass(t *testing.T) {
	sb := newTestSupabase()
	sc := "premium-ssd"
	sb.Spec.Database.Storage.StorageClassName = &sc
	pctx := newTestPlatformContext(sb)
	c := NewCNPGCluster(pctx)

	cluster := c.buildCluster()

	require.NotNil(t, cluster.Spec.StorageConfiguration.StorageClass)
	assert.Equal(t, "premium-ssd", *cluster.Spec.StorageConfiguration.StorageClass)
}
