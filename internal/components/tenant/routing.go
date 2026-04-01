package tenant

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const componentRouting = "routing"

// RoutingComponent manages the HTTPRoute for tenant traffic routing.
type RoutingComponent struct {
	ctx *components.TenantContext
}

func NewRoutingComponent(ctx *components.TenantContext) *RoutingComponent {
	return &RoutingComponent{ctx: ctx}
}

func (r *RoutingComponent) Name() string {
	return componentRouting
}

func (r *RoutingComponent) httpRouteName() string {
	return fmt.Sprintf("%s-routes", r.ctx.TenantID())
}

func ptrTo[T any](v T) *T {
	return &v
}

func (r *RoutingComponent) buildHTTPRoute() *gatewayv1.HTTPRoute {
	labels := resources.TenantLabels(r.ctx.InstanceName(), r.ctx.TenantID(), componentRouting)
	platformNS := r.ctx.Supabase.Namespace
	tenantNS := r.ctx.TenantNamespace
	tenantID := r.ctx.TenantID()

	hostname := gatewayv1.Hostname(fmt.Sprintf("%s.%s", tenantID, r.ctx.BaseDomain()))
	gatewayName := fmt.Sprintf("%s-gateway", r.ctx.Supabase.Name)
	pathPrefix := gatewayv1.PathMatchPathPrefix

	type routeTarget struct {
		pathPrefix  string
		serviceName string
		port        int32
	}

	targets := []routeTarget{
		{"/auth/v1", fmt.Sprintf("%s-auth", tenantID), 9999},
		{"/rest/v1", fmt.Sprintf("%s-rest", tenantID), 3000},
		{"/realtime/v1", fmt.Sprintf("%s-realtime", tenantID), 4000},
		{"/storage/v1", fmt.Sprintf("%s-storage", tenantID), 5000},
		{"/functions/v1", fmt.Sprintf("%s-functions", tenantID), 9000},
	}

	var rules []gatewayv1.HTTPRouteRule
	for _, t := range targets {
		rules = append(rules, gatewayv1.HTTPRouteRule{
			Matches: []gatewayv1.HTTPRouteMatch{
				{
					Path: &gatewayv1.HTTPPathMatch{
						Type:  &pathPrefix,
						Value: ptrTo(t.pathPrefix),
					},
				},
			},
			BackendRefs: []gatewayv1.HTTPBackendRef{
				{
					BackendRef: gatewayv1.BackendRef{
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Name:      gatewayv1.ObjectName(t.serviceName),
							Namespace: ptrTo(gatewayv1.Namespace(tenantNS)),
							Port:      ptrTo(gatewayv1.PortNumber(t.port)),
						},
					},
				},
			},
		})
	}

	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.httpRouteName(),
			Namespace: platformNS,
			Labels:    labels,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      gatewayv1.ObjectName(gatewayName),
						Namespace: ptrTo(gatewayv1.Namespace(platformNS)),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules:     rules,
		},
	}
}

func (r *RoutingComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if r.ctx.Tenant.Spec.Suspended {
		// Delete HTTPRoute when suspended
		return ctrl.Result{}, r.deleteHTTPRoute(ctx)
	}

	desired := r.buildHTTPRoute()
	existing := &gatewayv1.HTTPRoute{}
	err := r.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting HTTPRoute: %w", err)
	}

	if err != nil {
		if createErr := r.ctx.Client.Create(ctx, desired); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating HTTPRoute: %w", createErr)
		}
		r.ctx.Recorder.Eventf(r.ctx.Tenant, "Normal", "Created", "Created HTTPRoute %s", desired.Name)
		return ctrl.Result{}, nil
	}

	// Update
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
	if updateErr := r.ctx.Client.Update(ctx, existing); updateErr != nil {
		return ctrl.Result{}, fmt.Errorf("updating HTTPRoute: %w", updateErr)
	}

	return ctrl.Result{}, nil
}

func (r *RoutingComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	if r.ctx.Tenant.Spec.Suspended {
		return true, "suspended", nil
	}

	route := &gatewayv1.HTTPRoute{}
	key := client.ObjectKey{
		Namespace: r.ctx.Supabase.Namespace,
		Name:      r.httpRouteName(),
	}
	if err := r.ctx.Client.Get(ctx, key, route); err != nil {
		return false, "HTTPRoute not found", client.IgnoreNotFound(err)
	}

	return true, "HTTPRoute exists", nil
}

func (r *RoutingComponent) Finalize(ctx context.Context) error {
	return r.deleteHTTPRoute(ctx)
}

func (r *RoutingComponent) deleteHTTPRoute(ctx context.Context) error {
	route := &gatewayv1.HTTPRoute{}
	key := client.ObjectKey{
		Namespace: r.ctx.Supabase.Namespace,
		Name:      r.httpRouteName(),
	}
	if err := r.ctx.Client.Get(ctx, key, route); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.ctx.Client.Delete(ctx, route)
}
