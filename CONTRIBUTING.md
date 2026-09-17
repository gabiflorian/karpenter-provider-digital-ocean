# Contributing

Thanks for wanting to help. This provider is a personal project for DigitalOcean Kubernetes (DOKS).

## How to work

- Open an issue for bugs, questions, and design discussion.
- Open a pull request for code. Target `main` from a feature branch; do not push commits onto `main`.
- Keep PRs focused. Describe what you changed and how you tested it (unit tests, `kubectl` on a DOKS cluster, or both).

## Build and test

You need Go 1.26.6 (or a toolchain that can download it).

```bash
make tidy
make generate
make vet
make test
make build
```

Format Go with `gofmt` (or `go fmt ./...`). If you change `pkg/apis/`, run `make generate` and commit the CRD/DeepCopy output.

## Secrets

Never commit DigitalOcean tokens, kubeconfigs, or log files. `run.sh`, `run.sh.log`, and `run.sh.out` are gitignored. Put a PAT in `~/.config/doks-tocken.txt` (or `DOKS_TOKEN_FILE`) locally only.

## Karpenter core

This repo uses a patched Karpenter via a Go `replace` on [`gabiflorian/karpenter`](https://github.com/gabiflorian/karpenter) (`gabiflorian/digital-ocean` branch).

- Provider logic (DOKS node pools, `DONodeClass`, godo client) belongs **here**.
- Core registration / provisioning patches belong on **that fork**, not in `kubernetes-sigs/karpenter` and not copied into this tree.

If a core change is required, open a PR on the fork and bump the `replace` commit in `go.mod` / `go.sum`.

## License

Contributions are accepted under the Apache License 2.0. See [LICENSE](LICENSE).
