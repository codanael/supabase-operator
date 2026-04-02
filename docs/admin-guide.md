# Supabase Operator - Kubernetes Administrator Guide

This guide covers installation, configuration, and day-to-day operations of the Supabase Operator for Kubernetes administrators.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Creating a Supabase Instance](#creating-a-supabase-instance)
- [Creating Tenants](#creating-tenants)
- [Configuration Reference](#configuration-reference)
- [Storage Backends](#storage-backends)
- [Authentication Configuration](#authentication-configuration)
- [Backups](#backups)
- [Resource Presets](#resource-presets)
- [Monitoring](#monitoring)
- [Tenant Lifecycle Management](#tenant-lifecycle-management)
- [Uninstalling](#uninstalling)
- [Troubleshooting](#troubleshooting)

## Overview

The Supabase Operator is a Kubernetes operator that automates the deployment and management of [Supabase](https://supabase.com) infrastructure on Kubernetes. It uses a **two-tier architecture**:

1. **Supabase** (platform-level) - Manages shared infrastructure: a CloudNativePG PostgreSQL cluster, a Gateway API gateway, and optional platform services (Imgproxy, Studio, Analytics, Vector, Supavisor).
2. **SupabaseTenant** (tenant-level) - Provisions isolated tenants, each with their own database, Auth (GoTrue), PostgREST, Realtime, Storage, and Edge Functions services, plus HTTP routing.

Each tenant gets a dedicated Kubernetes namespace (`supabase-<tenantId>`), auto-generated secrets (JWT keys, database credentials), and an HTTP endpoint at `<tenantId>.<baseDomain>`.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Supabase (Platform)                   │
│                                                         │
│  ┌──────────────┐  ┌─────────┐  ┌───────────────────┐  │
│  │ CNPG Cluster │  │ Gateway │  │ Optional Services │  │
│  │ (PostgreSQL) │  │ (Envoy) │  │  Studio, Imgproxy │  │
│  │              │  │         │  │  Analytics, Vector │  │
│  │              │  │         │  │  Supavisor         │  │
│  └──────────────┘  └─────────┘  └───────────────────┘  │
└─────────────────────────────────────────────────────────┘
          │                │
          ▼                ▼
┌──────────────────┐ ┌──────────────────┐
│ SupabaseTenant A │ │ SupabaseTenant B │  ...
│ ns: supabase-a   │ │ ns: supabase-b   │
│                  │ │                  │
│  Auth (GoTrue)   │ │  Auth (GoTrue)   │
│  PostgREST       │ │  PostgREST       │
│  Realtime        │ │  Realtime        │
│  Storage         │ │  Storage         │
│  Functions       │ │  Functions       │
│  HTTPRoute       │ │  HTTPRoute       │
└──────────────────┘ └──────────────────┘
```

## Prerequisites

Before installing the operator, ensure the following are available in your cluster:

| Dependency | Minimum Version | Purpose |
|---|---|---|
| Kubernetes | v1.28+ | Cluster |
| [CloudNativePG](https://cloudnative-pg.io/) | v1.25+ | PostgreSQL database management |
| [Gateway API CRDs](https://gateway-api.sigs.k8s.io/) | v1.2+ | Ingress routing via HTTPRoute |
| A Gateway API implementation (e.g., [Envoy Gateway](https://gateway.envoyproxy.io/)) | v1.3+ | Traffic routing |

### Install prerequisites

```bash
# Install CloudNativePG operator
kubectl apply --server-side -f \
  "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-1.25.1.yaml"
kubectl wait --for=condition=Available deployment/cnpg-controller-manager \
  -n cnpg-system --timeout=120s

# Install Gateway API CRDs
kubectl apply -f \
  "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml"

# Install Envoy Gateway (or your preferred Gateway API implementation)
kubectl apply --server-side -f \
  "https://github.com/envoyproxy/gateway/releases/download/v1.3.0/install.yaml"
kubectl wait --for=condition=Available deployment/envoy-gateway \
  -n envoy-gateway-system --timeout=120s

# Create a GatewayClass
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy-gateway
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF
```

## Installation

### Option 1: Consolidated YAML installer

```bash
# Build the installer (from the repo)
make build-installer IMG=<your-registry>/supabase-operator:v0.0.1

# Apply it
kubectl apply -f dist/install.yaml
```

Or apply a published installer directly:

```bash
kubectl apply -f https://raw.githubusercontent.com/codanael/supabase-operator/<tag>/dist/install.yaml
```

### Option 2: Kustomize (step by step)

```bash
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/supabase-operator:v0.0.1

# Install CRDs
make install

# Deploy the operator
make deploy IMG=<your-registry>/supabase-operator:v0.0.1
```

### Verify the installation

```bash
# Check the operator pod is running
kubectl get pods -n supabase-operator-system

# Check the CRDs are registered
kubectl get crd supabases.supabase.codanael.io
kubectl get crd supabasetenants.supabase.codanael.io
```

## Creating a Supabase Instance

A `Supabase` resource provisions the shared platform infrastructure (database cluster + gateway).

```yaml
apiVersion: supabase.codanael.io/v1alpha1
kind: Supabase
metadata:
  name: main
  namespace: supabase-system
spec:
  database:
    instances: 3              # CNPG replicas (HA)
    imageName: supabase/postgres:15.8.1.085
    storage:
      size: 10Gi
      # storageClassName: gp3  # optional
    # resources:               # optional resource limits
    #   requests:
    #     cpu: 500m
    #     memory: 1Gi
  gateway:
    gatewayClassName: envoy-gateway
    baseDomain: supabase.example.com
    # tls:                     # optional TLS
    #   certificateSecretRef: supabase-tls
  # Optional platform services:
  # imgproxy:
  #   enabled: true
  #   replicas: 1
  # studio:
  #   enabled: true
  # analytics:
  #   enabled: true
  # vector:
  #   enabled: true
  # supavisor:
  #   enabled: true
```

```bash
kubectl apply -f supabase.yaml

# Watch status
kubectl get supabase -n supabase-system -w
```

The output shows printed columns for Phase, DB Ready, GW Ready, Tenant count, and Age:

```
NAME   PHASE          DB READY   GW READY   TENANTS   AGE
main   Provisioning   false      false      0         5s
main   Ready          true       true       0         45s
```

Wait until Phase is `Ready` before creating tenants.

## Creating Tenants

A `SupabaseTenant` provisions an isolated Supabase project with its own namespace, services, and credentials.

```yaml
apiVersion: supabase.codanael.io/v1alpha1
kind: SupabaseTenant
metadata:
  name: acme
  namespace: supabase-system    # same namespace as the parent Supabase
spec:
  tenantId: acme                # must match: ^[a-z][a-z0-9-]*[a-z0-9]$
  supabaseRef: main             # name of the parent Supabase resource
  auth:
    siteURL: https://app.acme.com
    email:
      enabled: true
      autoconfirm: false
    # disableSignup: false
    # smtp:
    #   host: smtp.example.com
    #   port: 587
    #   credentialsSecret: acme-smtp-credentials
    # external:
    #   google:
    #     enabled: true
    #     credentialsSecret: acme-google-oauth
  rest:
    schemas:
      - public
      - graphql_public
    maxRows: 1000
  storage:
    backend: file
    fileSizeLimit: 52428800      # 50 MB
    imageTransformation: true
  functions:
    verifyJWT: true
  resources: small               # small | medium | large | custom
```

```bash
kubectl apply -f tenant.yaml

# Watch tenant status
kubectl get supabasetenant -n supabase-system -w
```

```
NAME   TENANT ID   PHASE          ENDPOINT                       AGE
acme   acme        Provisioning                                  5s
acme   acme        Ready          acme.supabase.example.com      60s
```

The operator automatically:
- Creates namespace `supabase-acme`
- Generates JWT secrets (anon key, service role key) and database credentials
- Provisions a CNPG Database resource for the tenant
- Deploys Auth, PostgREST, Realtime, Storage, and Functions services
- Creates an HTTPRoute for `<tenantId>.<baseDomain>`

### Accessing tenant secrets

```bash
# JWT secrets (anon key, service role key)
kubectl get secret acme-jwt -n supabase-acme -o jsonpath='{.data.anon-key}' | base64 -d
kubectl get secret acme-jwt -n supabase-acme -o jsonpath='{.data.service-role-key}' | base64 -d

# Database credentials
kubectl get secret acme-db-credentials -n supabase-acme -o jsonpath='{.data.postgres-password}' | base64 -d
```

## Configuration Reference

### Supabase Spec

| Field | Type | Default | Description |
|---|---|---|---|
| `database.instances` | int | `3` | Number of CNPG PostgreSQL replicas |
| `database.imageName` | string | `supabase/postgres:15.8.1.085` | PostgreSQL container image |
| `database.storage.size` | string | `10Gi` | PVC size per database instance |
| `database.storage.storageClassName` | string | *(cluster default)* | Storage class for PVCs |
| `database.resources` | ResourceRequirements | - | CPU/memory for database pods |
| `database.backup` | BackupSpec | - | Scheduled backup configuration |
| `gateway.gatewayClassName` | string | **required** | GatewayClass to use |
| `gateway.baseDomain` | string | **required** | Base domain for tenant endpoints |
| `gateway.tls.certificateSecretRef` | string | - | TLS certificate Secret name |
| `imgproxy` | ServiceSpec | disabled | Image proxy service |
| `studio` | ServiceSpec | disabled | Supabase Studio dashboard |
| `analytics` | ServiceSpec | disabled | Analytics service |
| `vector` | ServiceSpec | disabled | Vector log collector |
| `supavisor` | ServiceSpec | disabled | Connection pooler |
| `images` | ImageOverrides | - | Custom container images for services |

### SupabaseTenant Spec

| Field | Type | Default | Description |
|---|---|---|---|
| `tenantId` | string | **required** | Unique tenant identifier (1-63 chars, lowercase alphanumeric + hyphens) |
| `supabaseRef` | string | **required** | Name of the parent Supabase resource |
| `suspended` | bool | `false` | Scale tenant workloads to zero |
| `resources` | ResourcePreset | `small` | Resource sizing: `small`, `medium`, `large`, or `custom` |
| `auth.siteURL` | string | - | Site URL for auth redirects |
| `auth.additionalRedirectURLs` | []string | - | Additional allowed redirect URLs |
| `auth.disableSignup` | bool | `false` | Disable new user signups |
| `auth.email.enabled` | bool | `true` | Enable email authentication |
| `auth.email.autoconfirm` | bool | `false` | Auto-confirm email signups |
| `auth.smtp` | SMTPSpec | - | SMTP configuration for auth emails |
| `auth.external` | OAuthProviders | - | OAuth providers (Google, GitHub, Azure) |
| `rest.schemas` | []string | `[public, graphql_public]` | Exposed PostgREST schemas |
| `rest.maxRows` | int | `1000` | Maximum rows per request |
| `storage.backend` | string | `file` | Storage backend: `file`, `s3`, or `obc` |
| `storage.fileSizeLimit` | int64 | `52428800` | Max upload size in bytes (default 50MB) |
| `storage.imageTransformation` | bool | `false` | Enable image transformations |
| `storage.s3` | S3Config | - | S3 storage configuration |
| `storage.objectBucket` | ObjectBucketSpec | - | OBC storage configuration |
| `functions.verifyJWT` | bool | `true` | Require JWT for edge functions |
| `functions.source.configMapRef` | string | - | ConfigMap with function source code |

## Storage Backends

Tenant storage supports three backends:

### File (default)

Uses a PVC for file storage. Suitable for development and single-node setups.

```yaml
storage:
  backend: file
```

### S3

Uses an S3-compatible object store. Create a Secret with your credentials first:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tenant-s3-credentials
  namespace: supabase-<tenantId>
type: Opaque
stringData:
  accessKeyId: "AKIAIOSFODNN7EXAMPLE"
  secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

```yaml
storage:
  backend: s3
  s3:
    bucket: my-tenant-bucket
    region: us-east-1
    endpoint: https://s3.amazonaws.com   # optional, for S3-compatible stores
    forcePathStyle: false                 # set true for MinIO
    credentialsSecret: tenant-s3-credentials
```

### ObjectBucketClaim (OBC)

Uses the Kubernetes ObjectBucketClaim API (e.g., Rook-Ceph, NooBaa):

```yaml
storage:
  backend: obc
  objectBucket:
    storageClassName: ceph-bucket
    bucketPrefix: supabase
```

## Authentication Configuration

### SMTP for email auth

Create a Secret with SMTP credentials, then reference it:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: acme-smtp-credentials
  namespace: supabase-system
type: Opaque
stringData:
  username: "noreply@acme.com"
  password: "smtp-password"
---
# In the tenant spec:
auth:
  smtp:
    host: smtp.example.com
    port: 587
    credentialsSecret: acme-smtp-credentials
    senderName: "Acme App"
```

### OAuth providers

Create a Secret with OAuth credentials, then enable the provider:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: acme-google-oauth
  namespace: supabase-system
type: Opaque
stringData:
  clientId: "1234567890.apps.googleusercontent.com"
  clientSecret: "GOCSPX-xxxxxxxxxxxx"
---
# In the tenant spec:
auth:
  external:
    google:
      enabled: true
      credentialsSecret: acme-google-oauth
    github:
      enabled: true
      credentialsSecret: acme-github-oauth
    azure:
      enabled: true
      credentialsSecret: acme-azure-oauth
```

## Backups

Configure automated database backups using CNPG's ScheduledBackup with S3 storage:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backup-s3-credentials
  namespace: supabase-system
type: Opaque
stringData:
  ACCESS_KEY_ID: "AKIAIOSFODNN7EXAMPLE"
  ACCESS_SECRET_KEY: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
---
# In the Supabase spec:
database:
  instances: 3
  storage:
    size: 10Gi
  backup:
    schedule: "0 2 * * *"                       # daily at 2 AM
    destinationPath: "s3://my-backups/supabase/"
    s3Credentials:
      secretRef: backup-s3-credentials
```

## Resource Presets

Tenants support resource sizing presets that apply CPU/memory limits to all tenant services:

| Preset | Description |
|---|---|
| `small` | Minimal resources for development/testing |
| `medium` | Moderate resources for staging or light production |
| `large` | Production-grade resources |
| `custom` | No preset applied; set resources per-component manually |

```yaml
spec:
  resources: medium
```

## Monitoring

The operator exposes Prometheus metrics on the controller's metrics endpoint:

| Metric | Type | Description |
|---|---|---|
| `supabase_tenants_total` | Gauge | Total number of SupabaseTenant resources |
| `supabase_tenants_ready` | Gauge | Tenants in Ready phase |
| `supabase_tenants_suspended` | Gauge | Suspended tenants |
| `supabase_reconcile_duration_seconds` | Histogram | Reconciliation loop duration (labels: `controller`, `result`) |

### Status conditions

Both CRDs expose fine-grained status conditions you can monitor:

**Supabase conditions:** `DatabaseReady`, `BackupReady`, `GatewayReady`, `ImgproxyReady`, `StudioReady`, `AnalyticsReady`, `VectorReady`, `SupavisorReady`

**SupabaseTenant conditions:** `NamespaceReady`, `SecretsReady`, `DatabaseReady`, `AuthReady`, `RESTReady`, `RealtimeReady`, `StorageReady`, `FunctionsReady`, `RoutingReady`

```bash
# Check conditions
kubectl get supabase main -n supabase-system -o jsonpath='{.status.conditions}' | jq .
kubectl get supabasetenant acme -n supabase-system -o jsonpath='{.status.conditions}' | jq .
```

## Tenant Lifecycle Management

### Suspend a tenant

Scale a tenant's workloads to zero without deleting it:

```bash
kubectl patch supabasetenant acme -n supabase-system \
  --type merge -p '{"spec":{"suspended":true}}'
```

### Resume a tenant

```bash
kubectl patch supabasetenant acme -n supabase-system \
  --type merge -p '{"spec":{"suspended":false}}'
```

### Delete a tenant

Deleting a `SupabaseTenant` triggers the finalizer, which cleans up routing, the tenant database, and the tenant namespace in reverse order:

```bash
kubectl delete supabasetenant acme -n supabase-system
```

### Secret rotation

The operator detects changes to tenant secrets via a hash annotation. If you need to rotate secrets, delete the existing secret and the operator will regenerate it on the next reconciliation cycle:

```bash
kubectl delete secret acme-jwt -n supabase-acme
kubectl delete secret acme-db-credentials -n supabase-acme
```

## Uninstalling

```bash
# Delete all tenants first
kubectl delete supabasetenants --all -n supabase-system

# Delete the Supabase platform instance
kubectl delete supabase --all -n supabase-system

# Remove the operator
make undeploy

# Remove CRDs
make uninstall
```

## Troubleshooting

### Supabase stuck in "Provisioning"

Check conditions for which component is not ready:

```bash
kubectl get supabase main -n supabase-system -o yaml
```

Common causes:
- **CNPG CRDs not installed**: The operator performs a preflight check. Ensure CloudNativePG is installed.
- **Gateway API CRDs not installed**: Ensure Gateway API standard CRDs and a GatewayClass exist.
- **Insufficient storage**: Check that PVCs are bound (`kubectl get pvc -n supabase-system`).

### Tenant stuck in "Provisioning"

Check tenant conditions:

```bash
kubectl describe supabasetenant acme -n supabase-system
```

Inspect the tenant namespace for pod issues:

```bash
kubectl get pods -n supabase-acme
kubectl describe pod <pod-name> -n supabase-acme
```

### SupabaseRef not found

The `SupabaseTenant` must reference an existing `Supabase` resource in the same namespace. Verify the name matches:

```bash
kubectl get supabase -n supabase-system
```

### RBAC errors

The operator requires cluster-level permissions. Ensure the deploying user has `cluster-admin` or review the RBAC manifests in `config/rbac/`.

### Events

The operator emits Kubernetes events for resource creation and errors:

```bash
kubectl get events -n supabase-system --sort-by=.lastTimestamp
kubectl get events -n supabase-acme --sort-by=.lastTimestamp
```
