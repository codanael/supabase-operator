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

const componentDatabase = "database"

type CNPGCluster struct {
	ctx *components.PlatformContext
}

func NewCNPGCluster(ctx *components.PlatformContext) *CNPGCluster {
	return &CNPGCluster{ctx: ctx}
}

func (c *CNPGCluster) Name() string {
	return componentDatabase
}

func (c *CNPGCluster) clusterName() string {
	return fmt.Sprintf("%s-db", c.ctx.InstanceName())
}

func (c *CNPGCluster) buildCluster() *cnpgv1.Cluster {
	sb := c.ctx.Supabase
	dbSpec := sb.Spec.Database

	labels := resources.PlatformLabels(c.ctx.InstanceName(), componentDatabase)

	cluster := &cnpgv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.clusterName(),
			Namespace: c.ctx.Namespace(),
			Labels:    labels,
		},
		Spec: cnpgv1.ClusterSpec{
			Instances: int(dbSpec.Instances),
			ImageName: dbSpec.ImageName,
			StorageConfiguration: cnpgv1.StorageConfiguration{
				Size:         dbSpec.Storage.Size,
				StorageClass: dbSpec.Storage.StorageClassName,
			},
			Resources: dbSpec.Resources,
		},
	}

	if dbSpec.Backup != nil {
		cluster.Spec.Backup = buildBackupConfig(dbSpec.Backup)
	}

	return cluster
}

func buildBackupConfig(spec *v1alpha1.BackupSpec) *cnpgv1.BackupConfiguration {
	cfg := &cnpgv1.BackupConfiguration{
		BarmanObjectStore: &barmanapi.BarmanObjectStoreConfiguration{
			DestinationPath: spec.DestinationPath,
		},
	}

	if spec.S3Credentials != nil {
		cfg.BarmanObjectStore.BarmanCredentials = barmanapi.BarmanCredentials{
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

func (c *CNPGCluster) Reconcile(ctx context.Context) (ctrl.Result, error) {
	desired := c.buildCluster()

	if err := controllerutil.SetControllerReference(c.ctx.Supabase, desired, c.ctx.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting owner reference: %w", err)
	}

	existing := &cnpgv1.Cluster{}
	err := c.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting CNPG cluster: %w", err)
	}

	if err != nil {
		// Not found - create
		if createErr := c.ctx.Client.Create(ctx, desired); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating CNPG cluster: %w", createErr)
		}
		c.ctx.Recorder.Eventf(c.ctx.Supabase, "Normal", "Created", "Created CNPG Cluster %s", desired.Name)
		return ctrl.Result{}, nil
	}

	// Update mutable fields using patch to avoid conflicts with mutating webhooks
	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec.Instances = desired.Spec.Instances
	existing.Spec.ImageName = desired.Spec.ImageName
	existing.Spec.StorageConfiguration = desired.Spec.StorageConfiguration
	existing.Spec.Resources = desired.Spec.Resources
	existing.Spec.Backup = desired.Spec.Backup
	existing.Labels = desired.Labels

	if patchErr := c.ctx.Client.Patch(ctx, existing, patch); patchErr != nil {
		return ctrl.Result{}, fmt.Errorf("patching CNPG cluster: %w", patchErr)
	}

	return ctrl.Result{}, nil
}

func (c *CNPGCluster) Healthcheck(ctx context.Context) (bool, string, error) {
	cluster := &cnpgv1.Cluster{}
	key := client.ObjectKey{Namespace: c.ctx.Namespace(), Name: c.clusterName()}
	if err := c.ctx.Client.Get(ctx, key, cluster); err != nil {
		return false, "CNPG Cluster not found", client.IgnoreNotFound(err)
	}

	for _, cond := range cluster.Status.Conditions {
		if cond.Type == string(cnpgv1.ConditionClusterReady) {
			if cond.Status == metav1.ConditionTrue {
				return true, "CNPG Cluster is ready", nil
			}
			return false, fmt.Sprintf("CNPG Cluster not ready: %s", cond.Message), nil
		}
	}

	return false, "CNPG Cluster ready condition not found", nil
}

func (c *CNPGCluster) Finalize(ctx context.Context) error {
	return nil
}
