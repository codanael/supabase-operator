package tenant

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const componentNamespace = "namespace"

// NamespaceComponent manages the tenant namespace and cross-tenant NetworkPolicy.
type NamespaceComponent struct {
	ctx *components.TenantContext
}

func NewNamespaceComponent(ctx *components.TenantContext) *NamespaceComponent {
	return &NamespaceComponent{ctx: ctx}
}

func (n *NamespaceComponent) Name() string {
	return componentNamespace
}

func (n *NamespaceComponent) buildNamespace() *corev1.Namespace {
	labels := resources.TenantLabels(n.ctx.InstanceName(), n.ctx.TenantID(), componentNamespace)

	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   n.ctx.TenantNamespace,
			Labels: labels,
		},
	}
}

func (n *NamespaceComponent) buildNetworkPolicy() *networkingv1.NetworkPolicy {
	labels := resources.TenantLabels(n.ctx.InstanceName(), n.ctx.TenantID(), componentNamespace)

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deny-cross-tenant",
			Namespace: n.ctx.TenantNamespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // selects all pods in namespace
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							// Allow from same namespace
							PodSelector: &metav1.LabelSelector{},
						},
						{
							// Allow from platform namespace
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": n.ctx.Supabase.Namespace,
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func (n *NamespaceComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	// Create namespace if not exists
	desiredNS := n.buildNamespace()
	existingNS := &corev1.Namespace{}
	err := n.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredNS), existingNS)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting namespace: %w", err)
	}
	if err != nil {
		if createErr := n.ctx.Client.Create(ctx, desiredNS); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating namespace: %w", createErr)
		}
		n.ctx.Recorder.Eventf(n.ctx.Tenant, "Normal", "Created", "Created Namespace %s", desiredNS.Name)
	}

	// Create NetworkPolicy if not exists
	desiredNP := n.buildNetworkPolicy()
	existingNP := &networkingv1.NetworkPolicy{}
	err = n.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredNP), existingNP)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting network policy: %w", err)
	}
	if err != nil {
		if createErr := n.ctx.Client.Create(ctx, desiredNP); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating network policy: %w", createErr)
		}
		n.ctx.Recorder.Eventf(n.ctx.Tenant, "Normal", "Created", "Created NetworkPolicy %s", desiredNP.Name)
	}

	return ctrl.Result{}, nil
}

func (n *NamespaceComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	ns := &corev1.Namespace{}
	key := client.ObjectKey{Name: n.ctx.TenantNamespace}
	if err := n.ctx.Client.Get(ctx, key, ns); err != nil {
		return false, "Namespace not found", client.IgnoreNotFound(err)
	}
	if ns.Status.Phase == corev1.NamespaceActive {
		return true, "Namespace is active", nil
	}
	return false, fmt.Sprintf("Namespace phase: %s", ns.Status.Phase), nil
}

func (n *NamespaceComponent) Finalize(ctx context.Context) error {
	ns := &corev1.Namespace{}
	key := client.ObjectKey{Name: n.ctx.TenantNamespace}
	if err := n.ctx.Client.Get(ctx, key, ns); err != nil {
		return client.IgnoreNotFound(err)
	}
	return n.ctx.Client.Delete(ctx, ns)
}
