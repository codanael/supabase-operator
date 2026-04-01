package platform

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const componentGateway = "gateway"

type Gateway struct {
	ctx *components.PlatformContext
}

func NewGateway(ctx *components.PlatformContext) *Gateway {
	return &Gateway{ctx: ctx}
}

func (g *Gateway) Name() string {
	return componentGateway
}

func (g *Gateway) gatewayName() string {
	return fmt.Sprintf("%s-gateway", g.ctx.InstanceName())
}

func (g *Gateway) buildGateway() *gatewayv1.Gateway {
	sb := g.ctx.Supabase
	gwSpec := sb.Spec.Gateway

	labels := resources.PlatformLabels(g.ctx.InstanceName(), componentGateway)

	wildcardHostname := gatewayv1.Hostname(fmt.Sprintf("*.%s", gwSpec.BaseDomain))

	listener := gatewayv1.Listener{
		Name:     "https",
		Port:     443,
		Protocol: gatewayv1.HTTPSProtocolType,
		Hostname: &wildcardHostname,
	}

	if gwSpec.TLS != nil {
		tlsMode := gatewayv1.TLSModeTerminate
		listener.TLS = &gatewayv1.ListenerTLSConfig{
			Mode: &tlsMode,
			CertificateRefs: []gatewayv1.SecretObjectReference{
				{
					Name: gatewayv1.ObjectName(gwSpec.TLS.CertificateSecretRef),
				},
			},
		}
	}

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.gatewayName(),
			Namespace: g.ctx.Namespace(),
			Labels:    labels,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gwSpec.GatewayClassName),
			Listeners:        []gatewayv1.Listener{listener},
		},
	}

	return gw
}

func (g *Gateway) Reconcile(ctx context.Context) (ctrl.Result, error) {
	desired := g.buildGateway()

	if err := controllerutil.SetControllerReference(g.ctx.Supabase, desired, g.ctx.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting owner reference: %w", err)
	}

	existing := &gatewayv1.Gateway{}
	err := g.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting Gateway: %w", err)
	}

	if err != nil {
		// Not found - create
		if createErr := g.ctx.Client.Create(ctx, desired); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating Gateway: %w", createErr)
		}
		g.ctx.Recorder.Eventf(g.ctx.Supabase, "Normal", "Created", "Created Gateway %s", desired.Name)
		return ctrl.Result{}, nil
	}

	// Update mutable fields
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels

	if updateErr := g.ctx.Client.Update(ctx, existing); updateErr != nil {
		return ctrl.Result{}, fmt.Errorf("updating Gateway: %w", updateErr)
	}

	return ctrl.Result{}, nil
}

func (g *Gateway) Healthcheck(ctx context.Context) (bool, string, error) {
	gw := &gatewayv1.Gateway{}
	key := client.ObjectKey{Namespace: g.ctx.Namespace(), Name: g.gatewayName()}
	if err := g.ctx.Client.Get(ctx, key, gw); err != nil {
		return false, "Gateway not found", client.IgnoreNotFound(err)
	}

	for _, cond := range gw.Status.Conditions {
		if cond.Type == string(gatewayv1.GatewayConditionAccepted) {
			if cond.Status == metav1.ConditionTrue {
				return true, "Gateway is accepted", nil
			}
			return false, fmt.Sprintf("Gateway not accepted: %s", cond.Message), nil
		}
	}

	return false, "Gateway accepted condition not found", nil
}

func (g *Gateway) Finalize(ctx context.Context) error {
	return nil
}
