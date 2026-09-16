GOARCH ?= $(shell go env GOARCH)
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.3
LDFLAGS ?= -ldflags=-X=sigs.k8s.io/karpenter/pkg/operator.Version=$(shell git describe --tags --always 2>/dev/null | cut -d"v" -f2)

DOKS_TOKEN_FILE ?= $(HOME)/.config/doks-tocken.txt
CLUSTER_NAME ?=
CLUSTER_ID ?=
KARPENTER_DIR := $(shell go list -m -f '{{.Dir}}' sigs.k8s.io/karpenter)

.PHONY: help build test vet tidy generate install-crds run

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build controller binary to bin/
	CGO_ENABLED=0 GOARCH=$(GOARCH) go build $(LDFLAGS) -o bin/karpenter-provider-digital-ocean ./cmd/controller

test: ## Run unit tests
	go test ./...

vet: ## go vet
	go vet ./...

tidy: ## go mod tidy
	go mod tidy

generate: ## CRDs + DeepCopy (controller-gen)
	$(CONTROLLER_GEN) \
		object:headerFile="hack/boilerplate.go.txt" \
		crd:allowDangerousTypes=true \
		paths="./pkg/apis/..." \
		output:crd:artifacts:config=pkg/apis/crds

install-crds: ## Apply Karpenter + DONodeClass CRDs to the current cluster
	kubectl apply -f $(KARPENTER_DIR)/pkg/apis/crds
	kubectl apply -f pkg/apis/crds

run: build install-crds ## Run controller using DIGITALOCEAN_TOKEN from DOKS_TOKEN_FILE
	@test -s "$(DOKS_TOKEN_FILE)" || (echo "missing token file $(DOKS_TOKEN_FILE)"; exit 1)
	@echo "running against context $$(kubectl config current-context)"
	@SYSTEM_NAMESPACE=kube-system \
	DISABLE_LEADER_ELECTION=true \
	CLUSTER_NAME="$(CLUSTER_NAME)" \
	CLUSTER_ID="$(CLUSTER_ID)" \
	DIGITALOCEAN_TOKEN="$$(tr -d '\n\r ' < "$(DOKS_TOKEN_FILE)")" \
	./bin/karpenter-provider-digital-ocean
