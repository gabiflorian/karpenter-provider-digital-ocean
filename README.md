# Karpenter Provider DigitalOcean

[![CI](https://github.com/gabiflorian/karpenter-provider-digital-ocean/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/gabiflorian/karpenter-provider-digital-ocean/actions/workflows/ci.yml?query=branch%3Amain)

Karpenter CloudProvider for [DigitalOcean Kubernetes (DOKS)](https://docs.digitalocean.com/products/kubernetes/).

Capacity is added through **DOKS node pools** (create a pool, then `PUT count`), not self-managed Droplets.

## Status

This is an **alpha / proof-of-concept**. It is a personal project, not an official DigitalOcean product, and it is **not ready for production**.

APIs, Helm charts, and behavior can change without notice. The controller creates and deletes DOKS node pools with your API token; you are responsible for cluster cost, data, and downtime. Use a throwaway cluster, keep node-pool CPU limits low, and run it at your own risk. The [Apache License 2.0](LICENSE) applies: the software is provided “AS IS”, without warranty.

## Build

Requires Go 1.26.6 (or a toolchain that can download it).

```bash
make generate
make tidy
make vet
make local_build
```

The binary is written to `bin/karpenter-provider-digital-ocean`.

`make build` compiles with [ko](https://ko.build) and pushes a `linux/amd64` image to Docker Hub (`docker.io/gflorian/karpenter-provider-digital-ocean` by default). Install `ko`, then log in once:

```bash
go install github.com/ko-build/ko@latest
# Hub Access Token (Write), not your account password
echo "$DOCKERHUB_TOKEN" | ko login index.docker.io -u gflorian --password-stdin
make build
```

Override the destination with `KO_DOCKER_REPO` and tags with `IMAGE_TAGS`.

This provider uses a patched Karpenter core from [`gabiflorian/karpenter`](https://github.com/gabiflorian/karpenter) (`gabiflorian/digital-ocean` branch) via a Go module `replace`.

## Install with Helm

Install CRDs, then the controller. The controller chart creates a Secret (`DIGITALOCEAN_TOKEN`) and a Deployment.

```bash
helm upgrade --install karpenter-crd charts/karpenter-crd --namespace kube-system
helm upgrade --install karpenter charts/karpenter --namespace kube-system \
  --set apiToken="${DIGITALOCEAN_TOKEN}" \
  --set settings.clusterName="${CLUSTER_NAME}"
```

Use `--set credentialsSecretRef=<secret-name>` if the token already lives in a Secret whose key is `DIGITALOCEAN_TOKEN`. Then apply `examples/donodeclass.yaml` and `examples/nodepool.yaml`.

## Run locally

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
