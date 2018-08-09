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

.PHONY: gendeepcopy

all: generate build images

depend:
	dep version || go get -u github.com/golang/dep/cmd/dep
	dep ensure

depend-update: 
	dep ensure -update

generate: gendeepcopy

gendeepcopy:
	go build -o $$GOPATH/bin/deepcopy-gen sigs.k8s.io/cluster-api-provider-ssh/vendor/k8s.io/code-generator/cmd/deepcopy-gen
	deepcopy-gen \
	  -i ./cloud/ssh/providerconfig,./cloud/ssh/providerconfig/v1alpha1 \
	  -O zz_generated.deepcopy \
	  -h boilerplate.go.txt

build: depend
	CGO_ENABLED=0 go install -a -ldflags '-extldflags "-static"' sigs.k8s.io/cluster-api-provider-ssh/cmd/cluster-controller
	CGO_ENABLED=0 go install -a -ldflags '-extldflags "-static"' sigs.k8s.io/cluster-api-provider-ssh/cmd/machine-controller

images: depend
	$(MAKE) -C cmd/cluster-controller image
	$(MAKE) -C cmd/machine-controller image

push: depend
	$(MAKE) -C cmd/cluster-controller push
	$(MAKE) -C cmd/machine-controller push

dev_push: 
	$(MAKE) -C cmd/cluster-controller dev_push
	$(MAKE) -C cmd/machine-controller dev_push

dev_push_cluster: depend
	$(MAKE) -C cmd/cluster-controller dev_push

dev_push_machine: depend
	$(MAKE) -C cmd/machine-controller dev_push

check: depend fmt vet

test:
	go test -race -cover ./cmd/... ./cloud/...

fmt:
	hack/verify-gofmt.sh

vet:
	go vet ./...

compile:
	mkdir -p ./bin
	go build -o ./bin/cluster-controller ./cmd/cluster-controller
	go build -o ./bin/machine-controller ./cmd/machine-controller
	go build -o ./bin/clusterctl ./clusterctl

clean:
	rm -rf ./bin

goimport:
	goimports -w $(shell git ls-files "**/*.go" "*.go" | grep -v -e "vendor" | xargs echo)
