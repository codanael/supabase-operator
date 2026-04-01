package tenant

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/credentials"
	"github.com/codanael/supabase-operator/internal/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const componentSecrets = "secrets"

// SecretComponent manages JWT keys and DB credentials for a tenant.
type SecretComponent struct {
	ctx *components.TenantContext
}

func NewSecretComponent(ctx *components.TenantContext) *SecretComponent {
	return &SecretComponent{ctx: ctx}
}

func (s *SecretComponent) Name() string {
	return componentSecrets
}

func (s *SecretComponent) jwtSecretName() string {
	return fmt.Sprintf("%s-jwt", s.ctx.TenantID())
}

func (s *SecretComponent) dbCredentialsSecretName() string {
	return fmt.Sprintf("%s-db-credentials", s.ctx.TenantID())
}

func (s *SecretComponent) buildJWTSecret() (*corev1.Secret, error) {
	labels := resources.TenantLabels(s.ctx.InstanceName(), s.ctx.TenantID(), componentSecrets)

	jwtSecret, err := credentials.GenerateHMACSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating JWT secret: %w", err)
	}

	anonKey, err := credentials.GenerateAnonKey(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generating anon key: %w", err)
	}

	serviceRoleKey, err := credentials.GenerateServiceRoleKey(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generating service role key: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.jwtSecretName(),
			Namespace: s.ctx.TenantNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"jwt-secret":       []byte(jwtSecret),
			"anon-key":         []byte(anonKey),
			"service-role-key": []byte(serviceRoleKey),
		},
	}, nil
}

func (s *SecretComponent) buildDBCredentialsSecret() (*corev1.Secret, error) {
	labels := resources.TenantLabels(s.ctx.InstanceName(), s.ctx.TenantID(), componentSecrets)

	postgresPassword, err := credentials.GeneratePassword(24)
	if err != nil {
		return nil, fmt.Errorf("generating postgres password: %w", err)
	}

	authenticatorPassword, err := credentials.GeneratePassword(24)
	if err != nil {
		return nil, fmt.Errorf("generating authenticator password: %w", err)
	}

	authAdminPassword, err := credentials.GeneratePassword(24)
	if err != nil {
		return nil, fmt.Errorf("generating auth admin password: %w", err)
	}

	storageAdminPassword, err := credentials.GeneratePassword(24)
	if err != nil {
		return nil, fmt.Errorf("generating storage admin password: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.dbCredentialsSecretName(),
			Namespace: s.ctx.TenantNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"postgres-password":      []byte(postgresPassword),
			"authenticator-password": []byte(authenticatorPassword),
			"auth-admin-password":    []byte(authAdminPassword),
			"storage-admin-password": []byte(storageAdminPassword),
		},
	}, nil
}

func (s *SecretComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	// JWT Secret
	jwtKey := client.ObjectKey{Namespace: s.ctx.TenantNamespace, Name: s.jwtSecretName()}
	existingJWT := &corev1.Secret{}
	err := s.ctx.Client.Get(ctx, jwtKey, existingJWT)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting JWT secret: %w", err)
	}

	if err != nil {
		// Not found — create
		jwtSecret, genErr := s.buildJWTSecret()
		if genErr != nil {
			return ctrl.Result{}, genErr
		}
		if createErr := s.ctx.Client.Create(ctx, jwtSecret); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating JWT secret: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Tenant, "Normal", "Created", "Created JWT Secret %s", jwtSecret.Name)
		existingJWT = jwtSecret
	}

	// Populate TenantContext from the existing (or just-created) secret
	s.ctx.JWTSecret = string(existingJWT.Data["jwt-secret"])
	s.ctx.AnonKey = string(existingJWT.Data["anon-key"])
	s.ctx.ServiceRoleKey = string(existingJWT.Data["service-role-key"])

	// DB Credentials Secret
	dbKey := client.ObjectKey{Namespace: s.ctx.TenantNamespace, Name: s.dbCredentialsSecretName()}
	existingDB := &corev1.Secret{}
	err = s.ctx.Client.Get(ctx, dbKey, existingDB)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting DB credentials secret: %w", err)
	}

	if err != nil {
		// Not found — create
		dbSecret, genErr := s.buildDBCredentialsSecret()
		if genErr != nil {
			return ctrl.Result{}, genErr
		}
		if createErr := s.ctx.Client.Create(ctx, dbSecret); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating DB credentials secret: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Tenant, "Normal", "Created", "Created DB Credentials Secret %s", dbSecret.Name)
		existingDB = dbSecret
	}

	// Populate DatabasePassword for downstream components
	s.ctx.DatabasePassword = string(existingDB.Data["postgres-password"])

	return ctrl.Result{}, nil
}

func (s *SecretComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	jwtKey := client.ObjectKey{Namespace: s.ctx.TenantNamespace, Name: s.jwtSecretName()}
	if err := s.ctx.Client.Get(ctx, jwtKey, &corev1.Secret{}); err != nil {
		return false, "JWT secret not found", client.IgnoreNotFound(err)
	}

	dbKey := client.ObjectKey{Namespace: s.ctx.TenantNamespace, Name: s.dbCredentialsSecretName()}
	if err := s.ctx.Client.Get(ctx, dbKey, &corev1.Secret{}); err != nil {
		return false, "DB credentials secret not found", client.IgnoreNotFound(err)
	}

	return true, "All secrets exist", nil
}

func (s *SecretComponent) Finalize(_ context.Context) error {
	// No-op: namespace deletion cascades all namespaced resources
	return nil
}
