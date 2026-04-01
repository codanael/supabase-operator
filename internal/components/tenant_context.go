package components

import (
	"fmt"
	"net/url"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TenantContext holds all contextual information needed to reconcile a tenant's components.
type TenantContext struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	Tenant          *v1alpha1.SupabaseTenant
	Supabase        *v1alpha1.Supabase
	TenantNamespace string // "supabase-{tenantId}"
	DatabaseName    string // tenantId
	DatabaseHost    string // "{supabase.Name}-db-rw.{supabase.Namespace}.svc.cluster.local"
	DatabasePort    string // "5432"

	// Populated by SecretComponent during reconciliation:
	JWTSecret      string
	AnonKey        string
	ServiceRoleKey string

	// Populated by DatabaseComponent during reconciliation:
	DatabasePassword string

	// Populated by SecretComponent during reconciliation:
	// Hash of all secret data for rotation detection.
	SecretHash string
}

// NewTenantContext creates a new TenantContext from the given resources.
func NewTenantContext(
	cl client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	tenant *v1alpha1.SupabaseTenant,
	supabase *v1alpha1.Supabase,
) *TenantContext {
	return &TenantContext{
		Client:          cl,
		Scheme:          scheme,
		Recorder:        recorder,
		Tenant:          tenant,
		Supabase:        supabase,
		TenantNamespace: fmt.Sprintf("supabase-%s", tenant.Spec.TenantID),
		DatabaseName:    tenant.Spec.TenantID,
		DatabaseHost:    fmt.Sprintf("%s-db-rw.%s.svc.cluster.local", supabase.Name, supabase.Namespace),
		DatabasePort:    "5432",
	}
}

// InstanceName returns the name of the parent Supabase resource.
func (c *TenantContext) InstanceName() string {
	return c.Supabase.Name
}

// TenantID returns the tenant identifier.
func (c *TenantContext) TenantID() string {
	return c.Tenant.Spec.TenantID
}

// BaseDomain returns the base domain from the parent Supabase gateway configuration.
func (c *TenantContext) BaseDomain() string {
	return c.Supabase.Spec.Gateway.BaseDomain
}

// DatabaseDSN returns a PostgreSQL connection string for the given user and password.
func (c *TenantContext) DatabaseDSN(user, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(password),
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
	)
}
