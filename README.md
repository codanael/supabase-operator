# Supabase Operator

A Kubernetes operator that automates the deployment and management of [Supabase](https://supabase.com) infrastructure using a multi-tenant architecture.

## Overview

The Supabase Operator manages two Custom Resources:

- **`Supabase`** - Provisions shared platform infrastructure: a [CloudNativePG](https://cloudnative-pg.io/) PostgreSQL cluster, a [Gateway API](https://gateway-api.sigs.k8s.io/) gateway, and optional services (Studio, Imgproxy, Analytics, Vector, Supavisor).
- **`SupabaseTenant`** - Provisions isolated tenants, each with their own namespace, database, Auth (GoTrue), PostgREST, Realtime, Storage, Edge Functions, and HTTP routing.

Each tenant gets:
- A dedicated namespace (`supabase-<tenantId>`)
- Auto-generated JWT keys and database credentials
- An HTTP endpoint at `<tenantId>.<baseDomain>`
- Independent lifecycle management (suspend, resume, delete)

## Prerequisites

- Kubernetes v1.28+
- [CloudNativePG](https://cloudnative-pg.io/) operator v1.25+
- [Gateway API CRDs](https://gateway-api.sigs.k8s.io/) v1.2+
- A Gateway API implementation (e.g., [Envoy Gateway](https://gateway.envoyproxy.io/) v1.3+)

See the [Administrator Guide](docs/admin-guide.md) for detailed prerequisite installation steps.

## Quick Start

### 1. Install the operator

```bash
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/supabase-operator:v0.0.1

# Install CRDs and deploy the operator
make install
make deploy IMG=<your-registry>/supabase-operator:v0.0.1
```

Or use the consolidated installer:

```bash
make build-installer IMG=<your-registry>/supabase-operator:v0.0.1
kubectl apply -f dist/install.yaml
```

### 2. Create a Supabase platform instance

```yaml
apiVersion: supabase.codanael.io/v1alpha1
kind: Supabase
metadata:
  name: main
  namespace: supabase-system
spec:
  database:
    instances: 3
    storage:
      size: 10Gi
  gateway:
    gatewayClassName: envoy-gateway
    baseDomain: supabase.example.com
```

### 3. Create a tenant

```yaml
apiVersion: supabase.codanael.io/v1alpha1
kind: SupabaseTenant
metadata:
  name: acme
  namespace: supabase-system
spec:
  tenantId: acme
  supabaseRef: main
  auth:
    siteURL: https://app.acme.com
    email:
      enabled: true
  storage:
    backend: file
  resources: small
```

```bash
kubectl apply -f supabase.yaml
kubectl apply -f tenant.yaml

# Watch status
kubectl get supabase,supabasetenant -n supabase-system -w
```

## Documentation

See the **[Administrator Guide](docs/admin-guide.md)** for complete documentation covering:

- Detailed installation and prerequisites
- Full configuration reference for both CRDs
- Storage backends (file, S3, ObjectBucketClaim)
- Authentication setup (email, SMTP, OAuth providers)
- Automated database backups
- Resource presets (small/medium/large)
- Prometheus metrics and monitoring
- Tenant lifecycle management (suspend/resume/delete)
- Troubleshooting

## Development

### Build and test

```bash
# Build the operator binary
make build

# Run unit tests
make test-unit

# Run integration tests (requires envtest)
make test

# Run E2E tests (requires kind)
make kind-up          # create kind cluster with CNPG + Gateway API
make kind-load        # load operator image into kind
make test-e2e         # run E2E tests
make kind-down        # tear down kind cluster
```

### Run locally

```bash
make install          # install CRDs
make run              # run the controller locally
```

## Uninstalling

```bash
kubectl delete supabasetenants --all -n supabase-system
kubectl delete supabase --all -n supabase-system
make undeploy
make uninstall
```

## Contributing

Contributions are welcome. Run `make help` for all available Make targets.

More information on the operator framework: [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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
