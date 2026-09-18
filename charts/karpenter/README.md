# Karpenter Provider DigitalOcean

This chart is **alpha / proof-of-concept**. It is not an official DigitalOcean product and is not ready for production. You are responsible for nodes the controller creates and for any cluster impact. See the [repository README](../../README.md).

Install the CRDs, then the controller. The controller chart creates a Secret for `DIGITALOCEAN_TOKEN` and a Deployment.

```bash
helm upgrade --install karpenter-crd charts/karpenter-crd --namespace kube-system
helm upgrade --install karpenter charts/karpenter --namespace kube-system \
  --set apiToken="${DIGITALOCEAN_TOKEN}" \
  --set settings.clusterName="${CLUSTER_NAME}"
```

To use a Secret you already created (key must be `DIGITALOCEAN_TOKEN`):

```bash
helm upgrade --install karpenter charts/karpenter --namespace kube-system \
  --set credentialsSecretRef=doks-credentials \
  --set settings.clusterName="${CLUSTER_NAME}"
```
