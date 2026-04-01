package platform

import (
	"testing"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledBackup_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewScheduledBackup(pctx)
	assert.Equal(t, "backup", c.Name())
}

func TestScheduledBackup_NilBackupSpec(t *testing.T) {
	sb := newTestSupabase()
	sb.Spec.Database.Backup = nil
	pctx := newTestPlatformContext(sb)
	c := NewScheduledBackup(pctx)

	// Healthcheck should return true when backup is not configured
	ready, msg, err := c.Healthcheck(nil)
	assert.NoError(t, err)
	assert.True(t, ready)
	assert.Equal(t, "backup not configured", msg)
}

func TestScheduledBackup_BuildScheduledBackup(t *testing.T) {
	sb := newTestSupabase()
	sb.Spec.Database.Backup = &v1alpha1.BackupSpec{
		Schedule:        "0 0 * * *",
		DestinationPath: "s3://my-bucket/backups",
		S3Credentials: &v1alpha1.S3CredentialsSpec{
			SecretRef: "backup-creds",
		},
	}
	pctx := newTestPlatformContext(sb)
	c := NewScheduledBackup(pctx)

	backup := c.buildScheduledBackup()

	assert.Equal(t, "main-db-backup", backup.Name)
	assert.Equal(t, "supabase-system", backup.Namespace)
	assert.Equal(t, "0 0 * * *", backup.Spec.Schedule)
	assert.Equal(t, "main-db", backup.Spec.Cluster.Name)
	assert.Equal(t, "self", backup.Spec.BackupOwnerReference)

	// Check labels
	assert.Equal(t, "supabase-operator", backup.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "supabase", backup.Labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "main", backup.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, "backup", backup.Labels["app.kubernetes.io/component"])
}

func TestScheduledBackup_BackupName(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewScheduledBackup(pctx)
	assert.Equal(t, "main-db-backup", c.backupName())
}

func TestScheduledBackup_ClusterName(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewScheduledBackup(pctx)
	assert.Equal(t, "main-db", c.clusterName())
}

func TestBuildBarmanConfig(t *testing.T) {
	spec := &v1alpha1.BackupSpec{
		DestinationPath: "s3://bucket/path",
		S3Credentials: &v1alpha1.S3CredentialsSpec{
			SecretRef: "my-creds",
		},
	}

	cfg := buildBarmanConfig(spec)

	assert.Equal(t, "s3://bucket/path", cfg.DestinationPath)
	require.NotNil(t, cfg.AWS)
	assert.Equal(t, "my-creds", cfg.AWS.AccessKeyIDReference.Name)
	assert.Equal(t, "ACCESS_KEY_ID", cfg.AWS.AccessKeyIDReference.Key)
	assert.Equal(t, "my-creds", cfg.AWS.SecretAccessKeyReference.Name)
	assert.Equal(t, "ACCESS_SECRET_KEY", cfg.AWS.SecretAccessKeyReference.Key)
}

func TestBuildBarmanConfig_NoS3(t *testing.T) {
	spec := &v1alpha1.BackupSpec{
		DestinationPath: "s3://bucket/path",
	}

	cfg := buildBarmanConfig(spec)

	assert.Equal(t, "s3://bucket/path", cfg.DestinationPath)
	assert.Nil(t, cfg.AWS)
}
