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

DEFAULT_VERSION := 0.0.0-dev
NMI_VERSION ?= $(DEFAULT_VERSION)
MIC_VERSION ?= $(DEFAULT_VERSION)
DEMO_VERSION ?= $(DEFAULT_VERSION)
IDENTITY_VALIDATOR_VERSION ?= $(DEFAULT_VERSION)

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

GO_BUILD_OPTIONS := --tags "netgo osusergo"  -ldflags "-s -X $(NMI_VERSION_VAR)=$(NMI_VERSION) -X $(MIC_VERSION_VAR)=$(MIC_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE) -extldflags '-static'"
E2E_TEST_OPTIONS := -count=1 -v -timeout 24h -ginkgo.progress $(E2E_TEST_OPTIONS_EXTRA)

# useful for other docker repos
REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= $(REGISTRY_NAME).azurecr.io
REPO_PREFIX ?= k8s/aad-pod-identity
NMI_IMAGE ?= $(REPO_PREFIX)/$(NMI_BINARY_NAME):$(NMI_VERSION)
MIC_IMAGE ?= $(REPO_PREFIX)/$(MIC_BINARY_NAME):$(MIC_VERSION)
DEMO_IMAGE ?= $(REPO_PREFIX)/$(DEMO_BINARY_NAME):$(DEMO_VERSION)
IDENTITY_VALIDATOR_IMAGE ?= $(REPO_PREFIX)/$(IDENTITY_VALIDATOR_BINARY_NAME):$(IDENTITY_VALIDATOR_VERSION)
ALL_DOCS := $(shell find . -name '*.md' -type f | sort)
TOOLS_MOD_DIR := ./tools
TOOLS_DIR := $(abspath ./.tools)

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(TOOLS_DIR)/misspell: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/misspell github.com/client9/misspell/cmd/misspell

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint $(TOOLS_DIR)/misspell
	$(TOOLS_DIR)/golangci-lint run
	$(TOOLS_DIR)/misspell -w $(ALL_DOCS) && \
	go mod tidy

.PHONY: clean-nmi
clean-nmi:
	rm -rf bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

.PHONY: clean-mic
clean-mic:
	rm -rf bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

.PHONY: clean-demo
clean-demo:
	rm -rf bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

.PHONY: clean-idenity-validator
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
	GOOS=$(GOOS) GOARCH=amd64 go build $(GO_BUILD_OPTIONS) -o "$(@)" "$(PKG_NAME)"

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

.PHONY: image-nmi
image-nmi:
	docker build -t "$(REGISTRY)/$(NMI_IMAGE)" --build-arg NMI_VERSION="$(NMI_VERSION)" --build-arg BASEIMAGE=us.gcr.io/k8s-artifacts-prod/build-image/debian-iptables-amd64:v12.1.0 --target=nmi .

.PHONY: image-mic
image-mic:
	docker build -t "$(REGISTRY)/$(MIC_IMAGE)" --build-arg MIC_VERSION="$(MIC_VERSION)" --target=mic .

.PHONY: image-demo
image-demo:
	docker build -t $(REGISTRY)/$(DEMO_IMAGE) --build-arg DEMO_VERSION="$(DEMO_VERSION)" --target=demo .

.PHONY: image-identity-validator
image-identity-validator:
	docker build -t $(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE) --build-arg IDENTITY_VALIDATOR_VERSION="$(IDENTITY_VALIDATOR_VERSION)" --target=identityvalidator .

.PHONY: image
image:image-nmi image-mic image-demo image-identity-validator

.PHONY: push-nmi
push-nmi: validate-version-NMI
	az acr repository show --name $(REGISTRY_NAME) --image $(NMI_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(NMI_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(NMI_IMAGE)

.PHONY: push-mic
push-mic: validate-version-MIC
	az acr repository show --name $(REGISTRY_NAME) --image $(MIC_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(MIC_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(MIC_IMAGE)

.PHONY: push-demo
push-demo: validate-version-DEMO
	az acr repository show --name $(REGISTRY_NAME) --image $(DEMO_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(DEMO_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(DEMO_IMAGE)

.PHONY: push-identity-validator
push-identity-validator: validate-version-IDENTITY_VALIDATOR
	az acr repository show --name $(REGISTRY_NAME) --image $(IDENTITY_VALIDATOR_IMAGE) > /dev/null 2>&1; if [ $$? -eq 0 ]; then echo "$(IDENTITY_VALIDATOR_IMAGE) already exists" && exit 0; fi
	docker push $(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE)

.PHONY: push
push: push-nmi push-mic push-demo push-identity-validator

.PHONY: e2e
e2e:
	make -C test/e2e/ run

.PHONY: unit-test
unit-test:
	GOOS=$(TEST_GOOS) go test -race -count=1 $(shell go list ./... | grep -v /test/e2e) -v

.PHONY: validate-version
validate-version: validate-version-NMI validate-version-MIC validate-version-IDENTITY_VALIDATOR validate-version-DEMO

.PHONY: validate-version-%
validate-version-%:
	@echo $(*)_VERSION=$($(*)_VERSION)
	@DEFAULT_VERSION=$(DEFAULT_VERSION) CHECK_VERSION="$($(*)_VERSION)" scripts/validate_version.sh

.PHONY: mod
mod:
	@go mod tidy

.PHONY: check-vendor
check-mod: mod
	@git diff --exit-code go.mod go.sum

.PHONY: helm-lint
helm-lint:
	# Download and install Helm
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
	# run lint on helm charts
	helm lint --strict charts/aad-pod-identity
