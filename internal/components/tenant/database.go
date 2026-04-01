package tenant

import (
	"context"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/database"
	"github.com/codanael/supabase-operator/internal/resources"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	componentDatabase    = "database"
	defaultPostgresImage = "supabase/postgres:15.8.1.085"
)

// DatabaseComponent manages the CNPG Database CR and init Job for a tenant.
type DatabaseComponent struct {
	ctx *components.TenantContext
}

func NewDatabaseComponent(ctx *components.TenantContext) *DatabaseComponent {
	return &DatabaseComponent{ctx: ctx}
}

func (d *DatabaseComponent) Name() string {
	return componentDatabase
}

func (d *DatabaseComponent) cnpgDatabaseName() string {
	return fmt.Sprintf("%s-db-%s", d.ctx.Supabase.Name, d.ctx.TenantID())
}

func (d *DatabaseComponent) initJobName() string {
	return fmt.Sprintf("%s-db-init-%s", d.ctx.Supabase.Name, d.ctx.TenantID())
}

func (d *DatabaseComponent) clusterName() string {
	return fmt.Sprintf("%s-db", d.ctx.Supabase.Name)
}

func (d *DatabaseComponent) buildCNPGDatabase() *cnpgv1.Database {
	labels := resources.TenantLabels(d.ctx.InstanceName(), d.ctx.TenantID(), componentDatabase)

	return &cnpgv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.cnpgDatabaseName(),
			Namespace: d.ctx.Supabase.Namespace,
			Labels:    labels,
		},
		Spec: cnpgv1.DatabaseSpec{
			Name:  d.ctx.TenantID(),
			Owner: "postgres",
			ClusterRef: corev1.LocalObjectReference{
				Name: d.clusterName(),
			},
		},
	}
}

func (d *DatabaseComponent) buildInitJob() (*batchv1.Job, error) {
	labels := resources.TenantLabels(d.ctx.InstanceName(), d.ctx.TenantID(), componentDatabase)

	// Get DB credentials from the context for init scripts
	dbCredsKey := client.ObjectKey{
		Namespace: d.ctx.TenantNamespace,
		Name:      fmt.Sprintf("%s-db-credentials", d.ctx.TenantID()),
	}
	_ = dbCredsKey // used conceptually — passwords come from context

	initSQL, err := database.CombinedInitSQL(database.InitParams{
		DatabaseName:          d.ctx.DatabaseName,
		JWTSecret:             d.ctx.JWTSecret,
		AuthenticatorPassword: d.ctx.DatabasePassword, // use the postgres password as default
		AuthAdminPassword:     d.ctx.DatabasePassword,
		StorageAdminPassword:  d.ctx.DatabasePassword,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering init SQL: %w", err)
	}

	image := d.ctx.Supabase.Spec.Database.ImageName
	if image == "" {
		image = defaultPostgresImage
	}

	backoffLimit := int32(3)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.initJobName(),
			Namespace: d.ctx.Supabase.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "init-db",
							Image: image,
							Command: []string{
								"psql",
								"-h", d.ctx.DatabaseHost,
								"-p", d.ctx.DatabasePort,
								"-U", "postgres",
								"-d", d.ctx.DatabaseName,
								"-c", initSQL,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PGPASSWORD",
									Value: d.ctx.DatabasePassword,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (d *DatabaseComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	// Create CNPG Database if not exists
	desiredDB := d.buildCNPGDatabase()
	existingDB := &cnpgv1.Database{}
	err := d.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredDB), existingDB)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting CNPG Database: %w", err)
	}
	if err != nil {
		if createErr := d.ctx.Client.Create(ctx, desiredDB); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating CNPG Database: %w", createErr)
		}
		d.ctx.Recorder.Eventf(d.ctx.Tenant, "Normal", "Created", "Created CNPG Database %s", desiredDB.Name)
	}

	// Create init Job if not exists
	desiredJob, jobErr := d.buildInitJob()
	if jobErr != nil {
		return ctrl.Result{}, jobErr
	}
	existingJob := &batchv1.Job{}
	err = d.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredJob), existingJob)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting init Job: %w", err)
	}
	if err != nil {
		if createErr := d.ctx.Client.Create(ctx, desiredJob); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating init Job: %w", createErr)
		}
		d.ctx.Recorder.Eventf(d.ctx.Tenant, "Normal", "Created", "Created init Job %s", desiredJob.Name)
	}

	return ctrl.Result{}, nil
}

func (d *DatabaseComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	// Check CNPG Database exists and is applied
	db := &cnpgv1.Database{}
	dbKey := client.ObjectKey{Namespace: d.ctx.Supabase.Namespace, Name: d.cnpgDatabaseName()}
	if err := d.ctx.Client.Get(ctx, dbKey, db); err != nil {
		return false, "CNPG Database not found", client.IgnoreNotFound(err)
	}

	if db.Status.Applied != nil && !*db.Status.Applied {
		return false, fmt.Sprintf("CNPG Database not applied: %s", db.Status.Message), nil
	}

	// Check init Job succeeded
	job := &batchv1.Job{}
	jobKey := client.ObjectKey{Namespace: d.ctx.Supabase.Namespace, Name: d.initJobName()}
	if err := d.ctx.Client.Get(ctx, jobKey, job); err != nil {
		return false, "Init Job not found", client.IgnoreNotFound(err)
	}

	if job.Status.Succeeded > 0 {
		return true, "Database initialized", nil
	}

	return false, fmt.Sprintf("Init Job not yet succeeded (active=%d, failed=%d)",
		job.Status.Active, job.Status.Failed), nil
}

func (d *DatabaseComponent) Finalize(ctx context.Context) error {
	// Delete CNPG Database (cross-namespace, not cascade-deleted)
	db := &cnpgv1.Database{}
	dbKey := client.ObjectKey{Namespace: d.ctx.Supabase.Namespace, Name: d.cnpgDatabaseName()}
	if err := d.ctx.Client.Get(ctx, dbKey, db); err == nil {
		if delErr := d.ctx.Client.Delete(ctx, db); delErr != nil {
			return fmt.Errorf("deleting CNPG Database: %w", delErr)
		}
	}

	// Delete init Job (cross-namespace resource)
	job := &batchv1.Job{}
	jobKey := client.ObjectKey{Namespace: d.ctx.Supabase.Namespace, Name: d.initJobName()}
	if err := d.ctx.Client.Get(ctx, jobKey, job); err == nil {
		if delErr := d.ctx.Client.Delete(ctx, job); delErr != nil {
			return fmt.Errorf("deleting init Job: %w", delErr)
		}
	}

	return nil
}
