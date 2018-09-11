ORG_PATH=github.com/Azure
PROJECT_NAME := aad-pod-identity
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"
NMI_BINARY_NAME := nmi
MIC_BINARY_NAME := mic
DEMO_BINARY_NAME := demo
NMI_VERSION=1.3
MIC_VERSION=1.2
DEMO_VERSION=1.2

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

GO_BUILD_OPTIONS := -buildmode=${GO_BUILD_MODE} -ldflags "-s -X $(VERSION_VAR)=$(NMI_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE)"

# useful for other docker repos
REGISTRY ?= nikhilbh
NMI_IMAGE_NAME := $(REGISTRY)/$(NMI_BINARY_NAME)
MIC_IMAGE_NAME := $(REGISTRY)/$(MIC_BINARY_NAME)
DEMO_IMAGE_NAME := $(REGISTRY)/$(DEMO_BINARY_NAME)

clean-nmi:
	rm -rf bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME)

clean-mic:
	rm -rf bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME)

clean-demo:
	rm -rf bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME)

clean:
	rm -rf bin/$(PROJECT_NAME)

build-nmi:clean-nmi
	go build -o bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(NMI_BINARY_NAME)

build-mic:clean-mic
	go build -o bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(MIC_BINARY_NAME)

build-demo:clean-demo
	go build -o bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(DEMO_BINARY_NAME)

build:clean
	go build -o bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(NMI_BINARY_NAME)
	go build -o bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(MIC_BINARY_NAME)
	go build -o bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME) $(GO_BUILD_OPTIONS) github.com/Azure/$(PROJECT_NAME)/cmd/$(DEMO_BINARY_NAME)

deepcopy-gen:
	deepcopy-gen -i ./pkg/apis/aadpodidentity/v1/ -o ../../../ -O aadpodidentity_deepcopy_generated -p aadpodidentity 

image-nmi:
	cp bin/$(PROJECT_NAME)/$(NMI_BINARY_NAME) images/nmi
	docker build -t $(NMI_IMAGE_NAME):$(NMI_VERSION) images/nmi

image-mic:
	cp bin/$(PROJECT_NAME)/$(MIC_BINARY_NAME) images/mic
	docker build -t $(MIC_IMAGE_NAME):$(MIC_VERSION) images/mic

image-demo:
	cp bin/$(PROJECT_NAME)/$(DEMO_BINARY_NAME) images/demo
	docker build -t $(DEMO_IMAGE_NAME):$(DEMO_VERSION) images/demo

image:image-nmi image-mic image-demo

push-nmi:
	docker push $(NMI_IMAGE_NAME):$(NMI_VERSION)

push-mic:
	docker push $(MIC_IMAGE_NAME):$(MIC_VERSION)

push-demo:
	docker push $(DEMO_IMAGE_NAME):$(DEMO_VERSION)

push:push-nmi push-mic push-demo

.PHONY: build
