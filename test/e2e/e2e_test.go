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

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codanael/supabase-operator/test/utils"
)

// namespace where the operator is deployed
const namespace = "supabase-operator-system"

// supabaseNamespace is the namespace where the Supabase platform resources live
const supabaseNamespace = "supabase-system"

// tenantNamespace is the namespace created for the test tenant
const tenantNamespace = "supabase-acme"

var _ = Describe("Supabase Operator E2E", Ordered, func() {
	var controllerPodName string

	// Collect debug info on failure
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events in supabase-system")
			cmd = exec.Command("kubectl", "get", "events", "-n", supabaseNamespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			}

			By("Fetching Kubernetes events in tenant namespace")
			cmd = exec.Command("kubectl", "get", "events", "-n", tenantNamespace, "--sort-by=.lastTimestamp")
			eventsOutput, err = utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Tenant namespace events:\n%s", eventsOutput)
			}
		}
	})

	// ---------------------------------------------------------------
	// Test 1: Operator Deployment
	// ---------------------------------------------------------------
	Context("Operator Deployment", func() {
		It("should have the controller-manager pod running", func() {
			verifyControllerUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp, 3*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	// ---------------------------------------------------------------
	// Test 2: Supabase Platform Creation
	// ---------------------------------------------------------------
	Context("Supabase Platform Creation", func() {
		BeforeAll(func() {
			By("applying the Supabase CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/supabase.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply Supabase CR")
		})

		AfterAll(func() {
			By("deleting the Supabase CR")
			cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/testdata/supabase.yaml", "--ignore-not-found")
			_, _ = utils.Run(cmd)
		})

		It("should create the CNPG Cluster", func() {
			verifyCNPGCluster := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "cluster", "-n", supabaseNamespace,
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get CNPG Cluster")
				g.Expect(output).NotTo(BeEmpty(), "No CNPG Cluster found")
			}
			Eventually(verifyCNPGCluster, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create the Gateway", func() {
			verifyGateway := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "gateway", "-n", supabaseNamespace,
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get Gateway")
				g.Expect(output).NotTo(BeEmpty(), "No Gateway found")
			}
			Eventually(verifyGateway, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create shared service Deployments", func() {
			sharedServices := []string{"imgproxy", "studio", "analytics", "vector", "supavisor"}
			for _, svc := range sharedServices {
				svc := svc
				verifyDeployment := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "deployment", "-n", supabaseNamespace,
						"-l", fmt.Sprintf("app.kubernetes.io/name=%s", svc),
						"-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred(), "Failed to get Deployment for %s", svc)
					g.Expect(output).NotTo(BeEmpty(), "No Deployment found for %s", svc)
				}
				Eventually(verifyDeployment, 5*time.Minute, 5*time.Second).Should(Succeed(),
					"Deployment for %s was not created", svc)
			}
		})

		It("should update the Supabase CR status phase", func() {
			verifyStatus := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "supabase", "e2e-test", "-n", supabaseNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get Supabase CR status")
				g.Expect(output).To(Or(
					Equal("Provisioning"),
					Equal("Ready"),
					Equal("Degraded"),
				), "Supabase CR phase should be Provisioning, Ready, or Degraded, got: %s", output)
			}
			Eventually(verifyStatus, 5*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	// ---------------------------------------------------------------
	// Test 3: Tenant Creation
	// ---------------------------------------------------------------
	Context("Tenant Lifecycle", func() {
		BeforeAll(func() {
			By("ensuring the Supabase platform CR exists")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/supabase.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply Supabase CR")

			// Wait for the platform to start reconciling before creating tenants
			time.Sleep(10 * time.Second)

			By("applying the SupabaseTenant CR")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/tenant.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply SupabaseTenant CR")
		})

		AfterAll(func() {
			By("deleting the SupabaseTenant CR")
			cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/testdata/tenant.yaml", "--ignore-not-found")
			_, _ = utils.Run(cmd)

			By("deleting the Supabase CR")
			cmd = exec.Command("kubectl", "delete", "-f", "test/e2e/testdata/supabase.yaml", "--ignore-not-found")
			_, _ = utils.Run(cmd)
		})

		It("should create the tenant namespace", func() {
			verifyNamespace := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "namespace", tenantNamespace,
					"-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Tenant namespace not found")
				g.Expect(output).To(Equal(tenantNamespace))
			}
			Eventually(verifyNamespace, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create the JWT secret in the tenant namespace", func() {
			verifySecret := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secret", "-n", tenantNamespace,
					"-l", "app.kubernetes.io/component=jwt",
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get JWT secret")
				g.Expect(output).NotTo(BeEmpty(), "No JWT secret found in tenant namespace")
			}
			Eventually(verifySecret, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create a CNPG Database in the platform namespace", func() {
			verifyDatabase := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "database", "-n", supabaseNamespace,
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get CNPG Database")
				g.Expect(output).NotTo(BeEmpty(), "No CNPG Database found")
			}
			Eventually(verifyDatabase, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create tenant service Deployments", func() {
			tenantServices := []string{"auth", "rest", "realtime", "storage", "functions"}
			for _, svc := range tenantServices {
				svc := svc
				verifyDeployment := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "deployment", "-n", tenantNamespace,
						"-l", fmt.Sprintf("app.kubernetes.io/name=%s", svc),
						"-o", "jsonpath={.items[*].metadata.name}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred(), "Failed to get Deployment for %s", svc)
					g.Expect(output).NotTo(BeEmpty(), "No Deployment found for %s", svc)
				}
				Eventually(verifyDeployment, 5*time.Minute, 5*time.Second).Should(Succeed(),
					"Deployment for %s was not created in tenant namespace", svc)
			}
		})

		It("should create an HTTPRoute in the platform namespace", func() {
			verifyHTTPRoute := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "httproute", "-n", supabaseNamespace,
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get HTTPRoute")
				g.Expect(output).NotTo(BeEmpty(), "No HTTPRoute found")
			}
			Eventually(verifyHTTPRoute, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should set the tenant status endpoint", func() {
			verifyEndpoint := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "supabasetenant", "acme", "-n", supabaseNamespace,
					"-o", "jsonpath={.status.endpoint}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get tenant status")
				g.Expect(output).NotTo(BeEmpty(), "Tenant endpoint not set")
			}
			Eventually(verifyEndpoint, 5*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	// ---------------------------------------------------------------
	// Test 4: Tenant Suspension
	// ---------------------------------------------------------------
	Context("Tenant Suspension", func() {
		BeforeAll(func() {
			By("ensuring platform and tenant exist")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/supabase.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/tenant.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			// Wait for tenant resources to be created
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "namespace", tenantNamespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("patching tenant to set suspended=true")
			cmd = exec.Command("kubectl", "patch", "supabasetenant", "acme",
				"-n", supabaseNamespace,
				"--type=merge",
				"-p", `{"spec":{"suspended":true}}`)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch tenant for suspension")
		})

		It("should scale all tenant Deployments to 0 replicas", func() {
			tenantServices := []string{"auth", "rest", "realtime", "storage", "functions"}
			for _, svc := range tenantServices {
				svc := svc
				verifyScaledDown := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "deployment", "-n", tenantNamespace,
						"-l", fmt.Sprintf("app.kubernetes.io/name=%s", svc),
						"-o", "jsonpath={.items[0].spec.replicas}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred(), "Failed to get Deployment replicas for %s", svc)
					g.Expect(output).To(Equal("0"), "Expected 0 replicas for %s, got %s", svc, output)
				}
				Eventually(verifyScaledDown, 3*time.Minute, 5*time.Second).Should(Succeed(),
					"Deployment %s was not scaled to 0", svc)
			}
		})

		It("should delete the HTTPRoute", func() {
			verifyHTTPRouteDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "httproute", "-n", supabaseNamespace,
					"-l", "app.kubernetes.io/instance=acme",
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(BeEmpty(), "HTTPRoute should be deleted after suspension")
			}
			Eventually(verifyHTTPRouteDeleted, 3*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	// ---------------------------------------------------------------
	// Test 5: Tenant Deletion
	// ---------------------------------------------------------------
	Context("Tenant Deletion", func() {
		BeforeAll(func() {
			By("ensuring platform and tenant exist (unsuspended)")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/supabase.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/testdata/tenant.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			// Unsuspend if still suspended from previous test
			cmd = exec.Command("kubectl", "patch", "supabasetenant", "acme",
				"-n", supabaseNamespace,
				"--type=merge",
				"-p", `{"spec":{"suspended":false}}`)
			_, _ = utils.Run(cmd)

			// Wait for tenant namespace to exist
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "namespace", tenantNamespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("deleting the SupabaseTenant CR")
			cmd = exec.Command("kubectl", "delete", "supabasetenant", "acme",
				"-n", supabaseNamespace, "--wait=false")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete SupabaseTenant CR")
		})

		AfterAll(func() {
			By("cleaning up the Supabase CR")
			cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/testdata/supabase.yaml", "--ignore-not-found")
			_, _ = utils.Run(cmd)
		})

		It("should delete the tenant namespace", func() {
			verifyNamespaceDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "namespace", tenantNamespace,
					"-o", "jsonpath={.metadata.name}")
				output, _ := utils.Run(cmd)
				g.Expect(output).To(BeEmpty(), "Tenant namespace should be deleted")
			}
			Eventually(verifyNamespaceDeleted, 5*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should delete the CNPG Database for the tenant", func() {
			verifyDatabaseDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "database", "-n", supabaseNamespace,
					"-l", "app.kubernetes.io/instance=acme",
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(BeEmpty(), "CNPG Database for tenant should be deleted")
			}
			Eventually(verifyDatabaseDeleted, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should delete the HTTPRoute for the tenant", func() {
			verifyHTTPRouteDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "httproute", "-n", supabaseNamespace,
					"-l", "app.kubernetes.io/instance=acme",
					"-o", "jsonpath={.items[*].metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(BeEmpty(), "HTTPRoute for tenant should be deleted")
			}
			Eventually(verifyHTTPRouteDeleted, 3*time.Minute, 5*time.Second).Should(Succeed())
		})
	})
})
