# Karpenter Provider DigitalOcean

Karpenter CloudProvider for [DigitalOcean Kubernetes (DOKS)](https://docs.digitalocean.com/products/kubernetes/).

Capacity is added through **DOKS node pools** (create a pool, then `PUT count`), not self-managed Droplets.

## Build

Requires Go 1.26.6 (or a toolchain that can download it).

```bash
make generate
make tidy
make vet
make build
```

The binary is written to `bin/karpenter-provider-digital-ocean`.

This provider uses a patched Karpenter core from [`gabiflorian/karpenter`](https://github.com/gabiflorian/karpenter) (`gabiflorian/digital-ocean` branch) via a Go module `replace`.

## Run

Place a DigitalOcean PAT in `~/.config/doks-tocken.txt` (or set `DOKS_TOKEN_FILE`). `make run` loads it into `DIGITALOCEAN_TOKEN` without printing it.

```bash
make run
```

This applies Karpenter + `DONodeClass` CRDs, then starts the controller against the current kubeconfig. Defaults:

| Variable | Default |
| --- | --- |
| `DOKS_TOKEN_FILE` | `~/.config/doks-tocken.txt` |
| `CLUSTER_NAME` | empty (optional; UUID is taken from the kubeconfig API server host) |
| `CLUSTER_ID` | empty (look up by name, or the only cluster on the token) |
| `SYSTEM_NAMESPACE` | `kube-system` |
| `DISABLE_LEADER_ELECTION` | `true` |

| Variable | Purpose |
| --- | --- |
| `CLUSTER_NAME` | Resolve the DOKS cluster UUID via `GET /v2/kubernetes/clusters` |
| `CLUSTER_ID` | Optional UUID override |
| `DIGITALOCEAN_TOKEN` | Personal access token for the DigitalOcean API |

Karpenter-managed pools are tagged `karpenter:managed` plus `karpenter:nodepool:<name>` and `karpenter:size:<slug>`. One pool per (NodePool, size slug); extra nodes scale `count`.
