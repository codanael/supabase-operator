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
	"time"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	supabasev1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	testutils "github.com/codanael/supabase-operator/test/utils"
)

var _ = Describe("Supabase Controller", func() {
	const (
		timeout  = 30 * time.Second
		interval = 250 * time.Millisecond
	)

	Context("When creating a Supabase resource", func() {
		const (
			resourceName = "test-sb"
			namespace    = "default"
		)

		var sb *supabasev1alpha1.Supabase

		BeforeEach(func() {
			sb = testutils.NewTestSupabase(resourceName, namespace)
			Expect(k8sClient.Create(ctx, sb)).To(Succeed())
		})

		AfterEach(func() {
			resource := &supabasev1alpha1.Supabase{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create a CNPG Cluster", func() {
			cluster := &cnpgv1.Cluster{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName + "-db",
					Namespace: namespace,
				}, cluster)
			}, timeout, interval).Should(Succeed())

			Expect(cluster.Spec.Instances).To(Equal(1))
			Expect(cluster.Spec.StorageConfiguration.Size).To(Equal("1Gi"))
		})

		It("should create a Gateway", func() {
			gw := &gatewayv1.Gateway{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName + "-gateway",
					Namespace: namespace,
				}, gw)
			}, timeout, interval).Should(Succeed())

			Expect(string(gw.Spec.GatewayClassName)).To(Equal("test-gateway-class"))
		})

		It("should create shared service Deployments", func() {
			deploymentNames := []string{
				resourceName + "-imgproxy",
				resourceName + "-analytics",
				resourceName + "-vector",
				resourceName + "-supavisor",
				resourceName + "-studio",
			}

			for _, name := range deploymentNames {
				deploy := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      name,
						Namespace: namespace,
					}, deploy)
				}, timeout, interval).Should(Succeed(), "Deployment %s should exist", name)
			}
		})

		It("should create Services for shared components", func() {
			serviceNames := []string{
				resourceName + "-imgproxy",
				resourceName + "-analytics",
				resourceName + "-vector",
				resourceName + "-supavisor",
				resourceName + "-studio",
			}

			for _, name := range serviceNames {
				svc := &corev1.Service{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      name,
						Namespace: namespace,
					}, svc)
				}, timeout, interval).Should(Succeed(), "Service %s should exist", name)
			}
		})

		It("should create a Vector ConfigMap", func() {
			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName + "-vector-config",
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			Expect(cm.Data).To(HaveKey("vector.yml"))
		})

		It("should set status phase after reconciliation", func() {
			Eventually(func() string {
				updated := &supabasev1alpha1.Supabase{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName,
					Namespace: namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).ShouldNot(BeEmpty())
		})
	})
})
