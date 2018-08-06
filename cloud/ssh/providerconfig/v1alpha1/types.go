// Copyright © 2018 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHMachineProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

	// A list of roles for this Machine to use.
	Roles []MachineRole `json:"roles,omitempty"`

	// ProvisionedMachineName is the binding reference to the Provisioned
	// Machine backing this Machine.
	ProvisionedMachineName string `json:"provisionedMachineName,omitempty"`

	// The data needed to ssh to the host
	SSHConfig SSHConfig `json:"sshConfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHClusterProviderConfig struct {
	metav1.TypeMeta `json:",inline"`
}

// The MachineRole indicates the purpose of the Machine, and will determine
// what software and configuration will be used when provisioning and managing
// the Machine. A single Machine may have more than one role, and the list and
// definitions of supported roles is expected to evolve over time.
type MachineRole string

const (
	MasterRole MachineRole = "Master"
	NodeRole   MachineRole = "Node"
	EtcdRole   MachineRole = "Etcd"
)

// SSHConfig specifies everything needed to ssh to a host
type SSHConfig struct {
	// The Username to use for the PrivateKey in secretName
	Username string `json:"username"`
	// The IP or hostname used to SSH to the machine
	Host string `json:"host"`
	// The Port used to SSH to the machine
	Port int `json:"port"`
	// The SSH public keys of the machine
	PublicKeys []string `json:"publicKeys,omitempty"`
	// The Secret with the username and private key used to SSH to the machine
	SecretName string `json:"secretName"`
}

type SSHMachineStatus string

const (
	MachineCreated SSHMachineStatus = "MachineCreated"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHMachineProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	Status          SSHMachineStatus `json:"status,omitempty"`
}

type SSHClusterStatus string

const (
	ClusterCreated SSHClusterStatus = "ClusterCreated"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHClusterProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	Status          SSHMachineStatus `json:"status,omitempty"`
}
