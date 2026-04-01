/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"time"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	supabasev1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	testutils "github.com/codanael/supabase-operator/test/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SupabaseTenant Controller", Ordered, func() {
	const (
		timeout  = 30 * time.Second
		interval = 250 * time.Millisecond

		supabaseName = "sb-tenant-test"
		tenantName   = "tenant-acme"
		namespace    = "default"
		tenantID     = "acme"
	)

	var (
		sb     *supabasev1alpha1.Supabase
		tenant *supabasev1alpha1.SupabaseTenant
	)

	BeforeAll(func() {
		// Create the parent Supabase resource first
		sb = testutils.NewTestSupabase(supabaseName, namespace)
		Expect(k8sClient.Create(ctx, sb)).To(Succeed())

		// Create the SupabaseTenant
		tenant = &supabasev1alpha1.SupabaseTenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tenantName,
				Namespace: namespace,
			},
			Spec: supabasev1alpha1.SupabaseTenantSpec{
				TenantID:    tenantID,
				SupabaseRef: supabaseName,
				Auth: supabasev1alpha1.TenantAuthSpec{
					SiteURL: "https://app.acme.com",
				},
				Resources: supabasev1alpha1.ResourcePresetSmall,
			},
		}
		Expect(k8sClient.Create(ctx, tenant)).To(Succeed())
	})

	AfterAll(func() {
		// Clean up tenant
		t := &supabasev1alpha1.SupabaseTenant{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: tenantName, Namespace: namespace}, t); err == nil {
			if controllerutil.ContainsFinalizer(t, tenantFinalizer) {
				controllerutil.RemoveFinalizer(t, tenantFinalizer)
				Expect(k8sClient.Update(ctx, t)).To(Succeed())
			}
			Expect(k8sClient.Delete(ctx, t)).To(Succeed())
		}

		// Clean up supabase
		s := &supabasev1alpha1.Supabase{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: supabaseName, Namespace: namespace}, s); err == nil {
			Expect(k8sClient.Delete(ctx, s)).To(Succeed())
		}
	})

	It("should have the finalizer present on the tenant CR", func() {
		Eventually(func() bool {
			t := &supabasev1alpha1.SupabaseTenant{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      tenantName,
				Namespace: namespace,
			}, t); err != nil {
				return false
			}
			return controllerutil.ContainsFinalizer(t, tenantFinalizer)
		}, timeout, interval).Should(BeTrue())
	})

	It("should create the tenant namespace", func() {
		ns := &corev1.Namespace{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: "supabase-acme"}, ns)
		}, timeout, interval).Should(Succeed())
	})

	It("should create the JWT secret in the tenant namespace", func() {
		secret := &corev1.Secret{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      "acme-jwt",
				Namespace: "supabase-acme",
			}, secret)
		}, timeout, interval).Should(Succeed())

		Expect(secret.Data).To(HaveKey("jwt-secret"))
		Expect(secret.Data).To(HaveKey("anon-key"))
		Expect(secret.Data).To(HaveKey("service-role-key"))
	})

	It("should create the DB credentials secret in the tenant namespace", func() {
		secret := &corev1.Secret{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      "acme-db-credentials",
				Namespace: "supabase-acme",
			}, secret)
		}, timeout, interval).Should(Succeed())

		Expect(secret.Data).To(HaveKey("postgres-password"))
	})

	It("should create the CNPG Database in the platform namespace", func() {
		db := &cnpgv1.Database{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-db-acme", supabaseName),
				Namespace: namespace,
			}, db)
		}, timeout, interval).Should(Succeed())

		Expect(db.Spec.Name).To(Equal("acme"))
		Expect(db.Spec.Owner).To(Equal("postgres"))
	})

	It("should create tenant service Deployments after DB init job succeeds", func() {
		// The controller gates service deployment on DB healthcheck.
		// Simulate the init job completing successfully.
		jobName := fmt.Sprintf("%s-db-init-acme", supabaseName)

		// Wait for the init job to be created first
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      jobName,
				Namespace: namespace,
			}, &batchv1.Job{})
		}, timeout, interval).Should(Succeed())

		// Also simulate the CNPG Database being applied
		Eventually(func() error {
			db := &cnpgv1.Database{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-db-acme", supabaseName),
				Namespace: namespace,
			}, db); err != nil {
				return err
			}
			applied := true
			db.Status.Applied = &applied
			return k8sClient.Status().Update(ctx, db)
		}, timeout, interval).Should(Succeed())

		// Update the Job status to simulate success
		Eventually(func() error {
			job := &batchv1.Job{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      jobName,
				Namespace: namespace,
			}, job); err != nil {
				return err
			}
			job.Status.Succeeded = 1
			return k8sClient.Status().Update(ctx, job)
		}, timeout, interval).Should(Succeed())

		// Now verify deployments are created
		deploymentNames := []string{
			"acme-auth",
			"acme-rest",
			"acme-realtime",
			"acme-storage",
			"acme-functions",
		}

		for _, name := range deploymentNames {
			deploy := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: "supabase-acme",
				}, deploy)
			}, timeout, interval).Should(Succeed(), "Deployment %s should exist", name)
		}
	})

	It("should create an HTTPRoute for routing", func() {
		// HTTPRoute should be created in the platform namespace after services are deployed
		route := &gatewayv1.HTTPRoute{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      "acme-routes",
				Namespace: namespace,
			}, route)
		}, timeout, interval).Should(Succeed())

		Expect(route.Spec.Hostnames).To(HaveLen(1))
	})

	It("should update status with namespace and endpoint", func() {
		Eventually(func() string {
			t := &supabasev1alpha1.SupabaseTenant{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      tenantName,
				Namespace: namespace,
			}, t); err != nil {
				return ""
			}
			return t.Status.Namespace
		}, timeout, interval).Should(Equal("supabase-acme"))

		Eventually(func() string {
			t := &supabasev1alpha1.SupabaseTenant{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(tenant), t); err != nil {
				return ""
			}
			return t.Status.Endpoint
		}, timeout, interval).Should(Equal("acme.supabase.test"))
	})
})
