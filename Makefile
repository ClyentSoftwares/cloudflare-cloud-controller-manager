VERSION ?= dev
IMAGE ?= ghcr.io/clyentsoftwares/cloudflare-cloud-controller-manager
GOLANGCI_LINT_VERSION ?= v2.12.2

LDFLAGS := -X 'github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cloudflare.providerVersion=$(VERSION)'

.PHONY: build test vet lint fmt docker-build

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/cloudflare-cloud-controller-manager .

test:
	go test ./... -race -count=1

vet:
	go vet ./...

# golangci-lint must be built with a toolchain at least as new as the go directive in go.mod
lint:
	GOTOOLCHAIN=go1.26.5 go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

fmt:
	gofmt -s -w .

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .
