ORG_PATH=github.com/Azure
PROJECT_NAME := aad-pod-identity
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"
NMI_BINARY_NAME := nmi
MIC_BINARY_NAME := mic
DEMO_BINARY_NAME := demo
IDENTITY_VALIDATOR_BINARY_NAME := identityvalidator
NMI_VERSION ?= 1.3
MIC_VERSION ?= 1.2
DEMO_VERSION ?= 1.2
IDENTITY_VALIDATOR_VERSION ?= 1.0

VERSION_VAR := $(REPO_PATH)/version.Version
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

GO_BUILD_OPTIONS := --tags "netgo osusergo"  -ldflags "-s -X $(VERSION_VAR)=$(NMI_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE) -extldflags '-static'"
E2E_TEST_OPTIONS := -count=1 -v

# useful for other docker repos
REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= upstreamk8sci.azurecr.io
REPO_PREFIX ?= k8s/aad-pod-identity
NMI_IMAGE ?= $(REPO_PREFIX)/$(NMI_BINARY_NAME):$(NMI_VERSION)
MIC_IMAGE ?= $(REPO_PREFIX)/$(MIC_BINARY_NAME):$(MIC_VERSION)
DEMO_IMAGE ?= $(REPO_PREFIX)/$(DEMO_BINARY_NAME):$(DEMO_VERSION)
IDENTITY_VALIDATOR_IMAGE ?= $(REPO_PREFIX)/$(IDENTITY_VALIDATOR_BINARY_NAME):$(IDENTITY_VALIDATOR_VERSION)

clean-nmi:
	rm -rf bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

clean-mic:
	rm -rf bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

clean-demo:
	rm -rf bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

clean-identity-validator:
	rm -rf bin/$(PROJECT_NAME)/$(IDENTITY_VALIDATOR_BINARY_NAME)

clean:
	rm -rf bin/$(PROJECT_NAME)

build-nmi:clean-nmi
	CGO_ENABLED=0 PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(NMI_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

build-mic:clean-mic
	CGO_ENABLED=0 PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(MIC_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

build-demo: build_tags := netgo osusergo
build-demo:clean-demo
	PKG_NAME=github.com/Azure/$(PROJECT_NAME)/cmd/$(DEMO_BINARY_NAME) ${MAKE} bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

bin/%:
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_OPTIONS) -o "$(@)" "$(PKG_NAME)"

build-identity-validator:clean-identity-validator
	PKG_NAME=github.com/Azure/$(PROJECT_NAME)/test/e2e/$(IDENTITY_VALIDATOR_BINARY_NAME) $(MAKE) bin/$(PROJECT_NAME)/$(IDENTITY_VALIDATOR_BINARY_NAME)

build:clean build-nmi build-mic build-demo build-identity-validator

deepcopy-gen:
	deepcopy-gen -i ./pkg/apis/aadpodidentity/v1/ -o ../../../ -O aadpodidentity_deepcopy_generated -p aadpodidentity

image-nmi:
	cp bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME) images/nmi
	docker build -t $(REGISTRY)/$(NMI_IMAGE) images/nmi

image-mic:
	cp bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME) images/mic
	docker build -t $(REGISTRY)/$(MIC_IMAGE) images/mic

image-demo:
	cp bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME) images/demo
	docker build -t $(REGISTRY)/$(DEMO_IMAGE) images/demo

image-identity-validator:
	cp bin/$(PROJECT_NAME)/$(IDENTITY_VALIDATOR_BINARY_NAME) images/identityvalidator
	docker build -t $(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE) images/identityvalidator

image:image-nmi image-mic image-demo image-identity-validator

push-nmi:
	az acr repository show --name $(REGISTRY_NAME) --image $(NMI_IMAGE) > /dev/null 2>&1; if [[ $$? -eq 0 ]]; then echo "$(NMI_IMAGE) already exists" && exit 1; fi
	docker push $(REGISTRY)/$(NMI_IMAGE)

push-mic:
	az acr repository show --name $(REGISTRY_NAME) --image $(MIC_IMAGE) > /dev/null 2>&1; if [[ $$? -eq 0 ]]; then echo "$(NMI_IMAGE) already exists" && exit 1; fi
	docker push $(REGISTRY)/$(MIC_IMAGE)

push-demo:
	az acr repository show --name $(REGISTRY_NAME) --image $(DEMO_IMAGE) > /dev/null 2>&1; if [[ $$? -eq 0 ]]; then echo "$(NMI_IMAGE) already exists" && exit 1; fi
	docker push $(REGISTRY)/$(DEMO_IMAGE)

push-identity-validator:
	az acr repository show --name $(REGISTRY_NAME) --image $(IDENTITY_VALIDATOR_IMAGE) > /dev/null 2>&1; if [[ $$? -eq 0 ]]; then echo "$(NMI_IMAGE) already exists" && exit 1; fi
	docker push $(REGISTRY)/$(IDENTITY_VALIDATOR_IMAGE)

push:push-nmi push-mic push-demo push-identity-validator

e2e:
	go test github.com/Azure/$(PROJECT_NAME)/test/e2e $(E2E_TEST_OPTIONS)

unit-test:
	go test $(shell go list ./... | grep -v /test/e2e) -v

.PHONY: build
