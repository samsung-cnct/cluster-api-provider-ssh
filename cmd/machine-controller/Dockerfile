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

ARG KUBEADM_VERSION=1.10.6

# Reproducible builder image
FROM golang:1.10.0 as builder
WORKDIR /go/src/github.com/samsung-cnct/cluster-api-provider-ssh
# This expects that the context passed to the docker build command is
# the cluster-api-provider-ssh directory.
# e.g. docker build -t <tag> -f <this_Dockerfile> <path_to_cluster-api-ssh>
COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-static"' github.com/samsung-cnct/cluster-api-provider-ssh/cmd/machine-controller

# Final container
FROM debian:stretch-slim
ARG KUBEADM_VERSION

RUN apt-get update && apt-get install -y ca-certificates openssh-server && rm -rf /var/lib/apt/lists/*

ADD https://storage.googleapis.com/kubernetes-release/release/v${KUBEADM_VERSION}/bin/linux/amd64/kubeadm /usr/bin/kubeadm

COPY --from=builder /go/bin/machine-controller .
