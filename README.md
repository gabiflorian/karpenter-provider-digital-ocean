# Karpenter Provider DigitalOcean

Skeleton controller for [Karpenter](https://sigs.k8s.io/karpenter) on [DigitalOcean Kubernetes (DOKS)](https://docs.digitalocean.com/products/kubernetes/).

This repository currently compiles and links a CloudProvider against Karpenter core. Create, Delete, Get, and List are stubs (`Unimplemented` / not-found). There is **no** DigitalOcean API client yet — this is not a working DOKS provisioner.

## Build

Requires Go 1.26.6 (or a toolchain that can download it). Direct dependencies match the AWS provider in this workspace: Karpenter `v1.14.1` (commit `2266468104f3`), controller-runtime `v0.24.1`, Kubernetes `v0.36.3`.

```bash
make generate
make tidy
make vet
make build
```

The binary is written to `bin/karpenter-provider-digital-ocean`.

```bash
go test ./...
```

## Environment variables

Parsed today, unused by the stubs. They will be required when node-pool provisioning is implemented:

| Variable | Purpose |
| --- | --- |
| `CLUSTER_NAME` | Resolve the DOKS cluster UUID via `GET /v2/kubernetes/clusters` |
| `CLUSTER_ID` | Optional UUID override (skip name lookup) |
| `DIGITALOCEAN_TOKEN` | Personal access token for the DigitalOcean API |

`make run` sets `SYSTEM_NAMESPACE` and `DISABLE_LEADER_ELECTION` so the process can start against a kubeconfig. A cluster is still required at runtime because Karpenter core builds a controller-runtime manager.

## Next phase

Fill the stubs using the **LKE node-pool model** (one DOKS pool per Karpenter NodePool + size slug, scale with `PUT count`), not self-managed Droplets joining the control plane.

See [`doks-api-research/doks-node-pools-and-self-managed-nodes.md`](../doks-api-research/doks-node-pools-and-self-managed-nodes.md).
