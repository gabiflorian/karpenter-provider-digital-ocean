GOARCH ?= $(shell go env GOARCH)
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.3
LDFLAGS ?= -ldflags=-X=sigs.k8s.io/karpenter/pkg/operator.Version=$(shell git describe --tags --always 2>/dev/null | cut -d"v" -f2)

.PHONY: help build test vet tidy generate run

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

run: ## Run controller against current kubeconfig (stubs only)
	SYSTEM_NAMESPACE=kube-system \
	DISABLE_LEADER_ELECTION=true \
	CLUSTER_NAME=$${CLUSTER_NAME} \
	go run ./cmd/controller/main.go
