ORG_PATH=github.com/Azure
PROJECT_NAME := aad-pod-identity
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"
NMI_BINARY_NAME := nmi
MIC_BINARY_NAME := mic
DEMO_BINARY_NAME := demo
SIMPLE_CMD_BINARY_NAME := simple
GOOS ?= linux
TEST_GOOS ?= linux
IDENTITY_VALIDATOR_BINARY_NAME := identityvalidator

DEFAULT_VERSION := v0.0.0-dev
IMAGE_VERSION ?= $(DEFAULT_VERSION)

NMI_VERSION_VAR := $(REPO_PATH)/version.NMIVersion
MIC_VERSION_VAR := $(REPO_PATH)/version.MICVersion
GIT_VAR := $(REPO_PATH)/version.GitCommit
BUILD_DATE_VAR := $(REPO_PATH)/version.BuildDate
BUILD_DATE := $$(date +%Y-%m-%d-%H:%M)
GIT_HASH := $$(git rev-parse --short HEAD)

ifeq ($(OS),Windows_NT)
	GO_BUILD_MODE = default
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S), Linux)
		GO_BUILD_MODE = pie
	endif
	ifeq ($(UNAME_S), Darwin)
		GO_BUILD_MODE = default
	endif
endif

GO_BUILD_OPTIONS := --tags "netgo osusergo"  -ldflags "-s -X $(NMI_VERSION_VAR)=$(IMAGE_VERSION) -X $(MIC_VERSION_VAR)=$(IMAGE_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE) -extldflags '-static'"
E2E_TEST_OPTIONS := -count=1 -v -timeout 24h -ginkgo.progress $(E2E_TEST_OPTIONS_EXTRA)

# useful for other docker repos
REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= k8s/aad-pod-identity
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
NMI_IMAGE := $(NMI_BINARY_NAME):$(IMAGE_VERSION)
MIC_IMAGE := $(MIC_BINARY_NAME):$(IMAGE_VERSION)
DEMO_IMAGE := $(DEMO_BINARY_NAME):$(IMAGE_VERSION)
IDENTITY_VALIDATOR_IMAGE := $(IDENTITY_VALIDATOR_BINARY_NAME):$(IMAGE_VERSION)
ALL_DOCS := $(shell find . -name '*.md' -type f | sort | grep -vE "website/(themes|node_modules)")
TOOLS_MOD_DIR := ./tools
TOOLS_DIR := $(abspath ./.tools)

# docker env var
DOCKER_BUILDKIT = 1
DOCKER_CLI_EXPERIMENTAL = enabled
export DOCKER_BUILDKIT DOCKER_CLI_EXPERIMENTAL
BUILD_PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7
# Output type of docker buildx build
OUTPUT_TYPE ?= registry

CONTROLLER_GEN := $(TOOLS_DIR)/controller-gen
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint
KUSTOMIZE := $(TOOLS_DIR)/kustomize
MISSPELL := $(TOOLS_DIR)/misspell

$(CONTROLLER_GEN): $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
		go build -o $(TOOLS_DIR)/controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen

$(GOLANGCI_LINT): $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(KUSTOMIZE): $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
		go build -o $(TOOLS_DIR)/kustomize sigs.k8s.io/kustomize/kustomize/v3

$(MISSPELL): $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/misspell github.com/client9/misspell/cmd/misspell

.PHONY: lint
lint: $(GOLANGCI_LINT) $(MISSPELL)
	$(GOLANGCI_LINT) run --timeout=5m
	$(MISSPELL) -w $(ALL_DOCS)
	$(MAKE) check-mod

.PHONY: clean-nmi
clean-nmi:
	rm -rf bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

.PHONY: clean-mic
clean-mic:
	rm -rf bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

.PHONY: clean-demo
clean-demo:
	rm -rf bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

.PHONY: clean-identity-validator
clean-identity-validator:
	rm -rf bin/$(PROJECT_NAME)/$(IDENTITY_VALIDATOR_BINARY_NAME)

.PHONY: clean-simple
clean-simple:
	rm -rf bin/$(PROJECT_NAME)/$(SIMPLE_CMD_BINARY_NAME)

.PHONY: clean
clean:
	rm -rf bin/$(PROJECT_NAME)

.PHONY: build-nmi
build-nmi: clean-nmi
	CGO_ENABLED=0 PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(NMI_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

.PHONY: build-mic
build-mic: clean-mic
	CGO_ENABLED=0 PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(MIC_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

.PHONY: build-simple
build-simple:
	CGO_ENABLED=0 PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(SIMPLE_CMD_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(SIMPLE_CMD_BINARY_NAME)

.PHONY: build-demo
build-demo: build_tags := netgo osusergo
build-demo: clean-demo
	PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(DEMO_BINARY_NAME) ${MAKE} bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

bin/%:
	GOOS=$(GOOS) go build $(GO_BUILD_OPTIONS) -o "$(@)" "$(PKG_NAME)"

.PHONY: build-identity-validator
build-identity-validator: clean-identity-validator
	PKG_NAME=github.com/Azure/$(PROJECT_NAME)/test/image/$(IDENTITY_VALIDATOR_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(IDENTITY_VALIDATOR_BINARY_NAME)

.PHONY: build
build: clean build-nmi build-mic build-demo build-identity-validator

.PHONY: precommit
precommit: build unit-test lint

.PHONY: deepcopy-gen
deepcopy-gen:
	deepcopy-gen -i ./pkg/apis/aadpodidentity/v1/ -o . -O aadpodidentity_deepcopy_generated -p aadpodidentity

.PHONY: docker-buildx-builder
docker-buildx-builder:
	docker run --rm --privileged docker/binfmt:820fdd95a9972a5308930a2bdfb8573dd4447ad3
	if ! docker buildx ls | grep -q container-builder; then \
		DOCKER_CLI_EXPERIMENTAL=enabled docker buildx create --name container-builder --use; \
	fi

.PHONY: image-nmi
image-nmi:
	docker buildx build \
		--target nmi \
		--no-cache \
		--build-arg IMAGE_VERSION=$(IMAGE_VERSION) \
		--platform "$(BUILD_PLATFORMS)" \
		--output=type=$(OUTPUT_TYPE) \
		-t $(REGISTRY)/$(NMI_IMAGE) .

.PHONY: image-mic
image-mic:
	docker buildx build \
		--target mic \
		--no-cache \
		--build-arg IMAGE_VERSION=$(IMAGE_VERSION) \
		--platform "$(BUILD_PLATFORMS)" \
		--output=type=$(OUTPUT_TYPE) \
		-t "$(REGISTRY)/$(MIC_IMAGE)" .

.PHONY: image-demo
image-demo:
	docker buildx build \
	 	--target demo \
		--no-cache \
	  	--build-arg IMAGE_VERSION=$(IMAGE_VERSION) \
		--platform "$(BUILD_PLATFORMS)" \
		--output=type=$(OUTPUT_TYPE) \
	    -t "$(REGISTRY)/$(DEMO_IMAGE)" .

.PHONY: image-identity-validator
image-identity-validator:
	docker buildx build \
	 	--target identityvalidator \
		--no-cache \
	 	--build-arg IMAGE_VERSION=$(IMAGE_VERSION) \
		--platform "$(BUILD_PLATFORMS)" \
		--output=type=$(OUTPUT_TYPE) \
	   	-t "$(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE)" .

.PHONY: images
images: image-nmi image-mic image-demo image-identity-validator

.PHONY: push-nmi
push-nmi: validate-version
	az acr repository show --name $(REGISTRY_NAME) --image $(NMI_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(NMI_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(NMI_IMAGE)

.PHONY: push-mic
push-mic: validate-version
	az acr repository show --name $(REGISTRY_NAME) --image $(MIC_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(MIC_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(MIC_IMAGE)

.PHONY: push-demo
push-demo: validate-version
	az acr repository show --name $(REGISTRY_NAME) --image $(DEMO_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(DEMO_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(DEMO_IMAGE)

.PHONY: push-identity-validator
push-identity-validator: validate-version
	az acr repository show --name $(REGISTRY_NAME) --image $(IDENTITY_VALIDATOR_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(IDENTITY_VALIDATOR_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE)

.PHONY: push
push: push-nmi push-mic push-demo push-identity-validator

.PHONY: e2e
e2e:
	make -C test/e2e/ run

.PHONY: unit-test
unit-test:
	GOOS=$(TEST_GOOS) CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic -count=1 $(shell go list ./... | grep -v /test/e2e) -v

.PHONY: validate-version
validate-version:
	@DEFAULT_VERSION=$(DEFAULT_VERSION) CHECK_VERSION="$(IMAGE_VERSION)" scripts/validate_version.sh

.PHONY: mod
mod:
	@go mod tidy

.PHONY: check-mod
check-mod: mod
	@git diff --exit-code go.mod go.sum

.PHONY: helm-lint
helm-lint:
	# Download and install Helm
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
	# run lint on helm charts
	helm lint --strict manifest_staging/charts/aad-pod-identity

.PHONY: generate-crds
generate-crds: $(CONTROLLER_GEN) $(KUSTOMIZE)
	$(CONTROLLER_GEN) crd:trivialVersions=true paths=./pkg/apis/aadpodidentity/v1/...
	$(KUSTOMIZE) build config/crd > config/crd/aadpodidentity.k8s.io.yaml
	rm -rf config/crd/aadpodidentity.k8s.io_*

.PHONY: promote-staging-manifest
promote-staging-manifest:
	@gsed -i "s/version: .*/version: ${NEW_CHART_VERSION}/g" manifest_staging/charts/aad-pod-identity/Chart.yaml
	@gsed -i "s/appVersion: .*/appVersion: ${NEW_APP_VERSION}/g" manifest_staging/charts/aad-pod-identity/Chart.yaml
	@gsed -i "s/tag: .*/tag: v${NEW_APP_VERSION}/g" manifest_staging/charts/aad-pod-identity/values.yaml
	@rm -rf deploy charts/aad-pod-identity
	@cp -r manifest_staging/deploy .
	@cp -r manifest_staging/charts .
	@mkdir -p ./charts/tmp
	@helm package ./charts/aad-pod-identity -d ./charts/tmp/
	@helm repo index ./charts/tmp --url https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts --merge ./charts/index.yaml
	@mv ./charts/tmp/* ./charts
	@rm -rf ./charts/tmp
