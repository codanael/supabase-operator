package platform

import (
	"context"
	"fmt"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const componentBackup = "backup"

// ScheduledBackupComponent manages a CNPG ScheduledBackup resource.
type ScheduledBackupComponent struct {
	ctx *components.PlatformContext
}

func NewScheduledBackup(ctx *components.PlatformContext) *ScheduledBackupComponent {
	return &ScheduledBackupComponent{ctx: ctx}
}

func (c *ScheduledBackupComponent) Name() string {
	return componentBackup
}

func (c *ScheduledBackupComponent) backupName() string {
	return fmt.Sprintf("%s-db-backup", c.ctx.InstanceName())
}

func (c *ScheduledBackupComponent) clusterName() string {
	return fmt.Sprintf("%s-db", c.ctx.InstanceName())
}

func (c *ScheduledBackupComponent) backupSpec() *v1alpha1.BackupSpec {
	return c.ctx.Supabase.Spec.Database.Backup
}

func (c *ScheduledBackupComponent) buildScheduledBackup() *cnpgv1.ScheduledBackup {
	spec := c.backupSpec()
	labels := resources.PlatformLabels(c.ctx.InstanceName(), componentBackup)

	sb := &cnpgv1.ScheduledBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.backupName(),
			Namespace: c.ctx.Namespace(),
			Labels:    labels,
		},
		Spec: cnpgv1.ScheduledBackupSpec{
			Schedule: spec.Schedule,
			Cluster: machineryapi.LocalObjectReference{
				Name: c.clusterName(),
			},
			BackupOwnerReference: "self",
			Method:               cnpgv1.BackupMethodBarmanObjectStore,
		},
	}

	return sb
}

func buildBarmanConfig(spec *v1alpha1.BackupSpec) *barmanapi.BarmanObjectStoreConfiguration {
	cfg := &barmanapi.BarmanObjectStoreConfiguration{
		DestinationPath: spec.DestinationPath,
	}

	if spec.S3Credentials != nil {
		cfg.BarmanCredentials = barmanapi.BarmanCredentials{
			AWS: &barmanapi.S3Credentials{
				AccessKeyIDReference: &machineryapi.SecretKeySelector{
					LocalObjectReference: machineryapi.LocalObjectReference{
						Name: spec.S3Credentials.SecretRef,
					},
					Key: "ACCESS_KEY_ID",
				},
				SecretAccessKeyReference: &machineryapi.SecretKeySelector{
					LocalObjectReference: machineryapi.LocalObjectReference{
						Name: spec.S3Credentials.SecretRef,
					},
					Key: "ACCESS_SECRET_KEY",
				},
			},
		}
	}

	return cfg
}

func (c *ScheduledBackupComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if c.backupSpec() == nil {
		return ctrl.Result{}, nil
	}

	desired := c.buildScheduledBackup()

	if err := controllerutil.SetControllerReference(c.ctx.Supabase, desired, c.ctx.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting owner reference on ScheduledBackup: %w", err)
	}

	existing := &cnpgv1.ScheduledBackup{}
	err := c.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting ScheduledBackup: %w", err)
	}

	if err != nil {
		// Not found - create
		if createErr := c.ctx.Client.Create(ctx, desired); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating ScheduledBackup: %w", createErr)
		}
		c.ctx.Recorder.Eventf(c.ctx.Supabase, "Normal", "Created", "Created ScheduledBackup %s", desired.Name)
		return ctrl.Result{}, nil
	}

	// Update mutable fields
	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec.Schedule = desired.Spec.Schedule
	existing.Spec.Cluster = desired.Spec.Cluster
	existing.Spec.Method = desired.Spec.Method
	existing.Labels = desired.Labels

	if patchErr := c.ctx.Client.Patch(ctx, existing, patch); patchErr != nil {
		return ctrl.Result{}, fmt.Errorf("patching ScheduledBackup: %w", patchErr)
	}

	return ctrl.Result{}, nil
}

func (c *ScheduledBackupComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	if c.backupSpec() == nil {
		return true, "backup not configured", nil
	}

	sb := &cnpgv1.ScheduledBackup{}
	key := client.ObjectKey{Namespace: c.ctx.Namespace(), Name: c.backupName()}
	if err := c.ctx.Client.Get(ctx, key, sb); err != nil {
		return false, "ScheduledBackup not found", client.IgnoreNotFound(err)
	}

	return true, "ScheduledBackup exists", nil
}

func (c *ScheduledBackupComponent) Finalize(_ context.Context) error {
	return nil
}
