IMG ?= localhost:32000/scada/operator:latest

# Run `go mod tidy` first to populate go.sum after checkout.
.PHONY: deps
deps:
	go mod tidy

# Re-generate deepcopy and CRD manifests from Go type annotations.
.PHONY: generate
generate:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	controller-gen crd rbac:roleName=scada-operator-role paths="./..." \
		output:crd:artifacts:config=../config/crd/bases \
		output:rbac:artifacts:config=../config/rbac

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build: deps fmt vet
	go build -o bin/manager ./cmd/main.go

.PHONY: test
test: deps fmt vet
	go test ./... -coverprofile cover.out

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)
