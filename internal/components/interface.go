package components

import (
	"context"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	Name() string
	Reconcile(ctx context.Context) (ctrl.Result, error)
	Healthcheck(ctx context.Context) (bool, string, error)
	Finalize(ctx context.Context) error
}

type PlatformContext struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Supabase *v1alpha1.Supabase
}

func (c *PlatformContext) Namespace() string {
	return c.Supabase.Namespace
}

func (c *PlatformContext) InstanceName() string {
	return c.Supabase.Name
}
