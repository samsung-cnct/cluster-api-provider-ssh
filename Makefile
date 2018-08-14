# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

.PHONY: gendeepcopy

all: generate build images ## Create generated code, build binaries and docker images

depend: ## Ensure dependencies are consistent with Gopkg.toml
	dep version || go get -u github.com/golang/dep/cmd/dep
	dep ensure

depend-update: ## Update dependencies to the latest allowed by Gopkg.toml
	dep ensure -update

generate: gendeepcopy ## Create generated code

gendeepcopy:
	go build -o $$GOPATH/bin/deepcopy-gen sigs.k8s.io/cluster-api-provider-ssh/vendor/k8s.io/code-generator/cmd/deepcopy-gen
	deepcopy-gen \
	  -i ./cloud/ssh/providerconfig,./cloud/ssh/providerconfig/v1alpha1 \
	  -O zz_generated.deepcopy \
	  -h boilerplate.go.txt

build: ## Build the cluster-controller and machine-controller binaries and install them in $GOPATH/bin
	CGO_ENABLED=0 go install -a -ldflags '-extldflags "-static"' sigs.k8s.io/cluster-api-provider-ssh/cmd/cluster-controller
	CGO_ENABLED=0 go install -a -ldflags '-extldflags "-static"' sigs.k8s.io/cluster-api-provider-ssh/cmd/machine-controller

images: ## Make cluster-controller and machine-controller images
	$(MAKE) -C cmd/cluster-controller image
	$(MAKE) -C cmd/machine-controller image

push: ## Push cluster-controller and machine-controller images to the image registry
	$(MAKE) -C cmd/cluster-controller push
	$(MAKE) -C cmd/machine-controller push

dev_push: dev_push_cluster dev_push_machine ## Push the development tagged cluster-controller and machine-controller images to the registry

dev_push_cluster: ## Push the development tagged cluster-controller image to the registry
	$(MAKE) -C cmd/cluster-controller dev_push

dev_push_machine: ## Push the development tagged machine-controller image to the registry
	$(MAKE) -C cmd/machine-controller dev_push

check: fmt vet ## Do go linting

test: ## Run tests
	go test -race -cover ./cmd/... ./cloud/...

fmt:
	hack/verify-gofmt.sh

vet:
	go vet ./...

compile: ## Compile the binaries into the ./bin directory
	mkdir -p ./bin
	go build -o ./bin/cluster-controller ./cmd/cluster-controller
	go build -o ./bin/machine-controller ./cmd/machine-controller
	go build -o ./bin/clusterctl ./clusterctl

clean: ## Clean up built files
	rm -rf ./bin

goimport: ## Run goimports
	goimports -w $(shell git ls-files "**/*.go" "*.go" | grep -v -e "vendor" | xargs echo)
