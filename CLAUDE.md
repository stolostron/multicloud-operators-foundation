# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Testing
- `make` or `make build` - Build all binaries
- `make images` - Build container image `quay.io/stolostron/multicloud-manager:latest`
- `make test-unit` - Run unit tests (requires kubebuilder)
- `make verify` - Run verification checks (CRDs and code generation)
- `make test` - Run all tests

### Pre-commit Checks
Before submitting a PR, always run:
```bash
make verify
make test
```

### Local Development Build
```bash
export BUILD_LOCALLY=1
make
```

### Deployment Commands
- `make deploy-foundation` - Deploy foundation components on hub and managed clusters
- `make clean-foundation` - Clean up foundation deployment
- `make deploy-hub` - Deploy hub components only
- `make deploy-klusterlet` - Deploy klusterlet components only
- `make deploy-addons` - Deploy addon components

### Code Generation
- `make update` - Update CRDs and generated code
- `make update-crds` - Update CRDs only
- `make update-scripts` - Update generated code only

## Architecture Overview

This is the **stolostron Foundation** project, which provides foundational components for Advanced Cluster Management (ACM) based on ManagedCluster resources.

### Main Components

The project consists of four main binary components:

1. **Controller** (`cmd/controller/`) - Hub-side controller that manages foundation resources
2. **Webhook** (`cmd/webhook/`) - Admission and validation webhook server
3. **ProxyServer** (`cmd/proxyserver/`) - Proxy server for cluster communication
4. **Agent** (`cmd/agent/`) - Agent running on managed clusters

### Key Package Structure

- `pkg/controllers/` - Contains all controller implementations:
  - `clusterset/` - ClusterSet-related controllers (syncclusterrolebinding, clusterdeployment, clustersetmapper)
  - Other controllers for managing foundation resources
- `pkg/webhook/` - Webhook implementations:
  - `validating/` - Validation webhooks
  - `mutating/` - Mutation webhooks
  - `serve/` - Webhook server implementation
- `pkg/cache/` - Caching mechanisms and watchers
- `pkg/helpers/` - Utility functions and helpers
- `pkg/utils/` - Common utilities
- `pkg/klusterlet/` - Klusterlet-related functionality
- `pkg/addon/` - Addon management functionality

### Deployment Structure

- `deploy/foundation/` - Foundation component deployments using Kustomize
- `deploy/managedcluster/` - Managed cluster component deployments

### Prerequisites

Before deploying Foundation components:
1. Install **Cluster Manager** and **Klusterlet** (see [Open Cluster Management](https://open-cluster-management.io))
2. Approve CSRs and accept managed clusters on the hub

### Technology Stack

- **Go 1.23.6** - Primary language
- **Kubernetes controller-runtime** - Controller framework
- **Open Cluster Management APIs** - Core APIs for multi-cluster management
- **Kustomize** - Deployment configuration management
- **Kubebuilder** - For testing infrastructure
- **Ginkgo/Gomega** - Testing framework

### Build System

Uses OpenShift build-machinery-go for:
- Golang builds and tooling
- Image building with imagebuilder
- Kustomize integration
- Binary data generation

The project supports both local development and CI/CD pipeline builds with container engines (podman/docker).