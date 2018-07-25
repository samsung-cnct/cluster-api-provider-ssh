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

package providerconfig

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SSHConfig specifies everything needed to ssh to a host
type SSHConfig struct {
	Username   string   `json:"username"`             // The Username to use for the PrivateKey in secretName
	Host       string   `json:"host"`                 // The IP or hostname used to SSH to the machine
	Port       int      `json:"port"`                 // The Port used to SSH to the machine
	PublicKeys []string `json:"publicKeys,omitempty"` // The SSH public keys of the machine
	SecretName string   `json:"secretName"`           // The Secret with the username and private key used to SSH to the machine
}

// MachineRole indicates the purpose of the Machine, and will determine
// what software and configuration will be used when provisioning and managing
// the Machine. A single Machine may have more than one role, and the list and
// definitions of supported roles is expected to evolve over time.
type MachineRole string

const (
	MasterRole MachineRole = "Master"
	NodeRole   MachineRole = "Node"
	EtcdRole   MachineRole = "Etcd"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHMachineProviderConfig struct {
	metav1.TypeMeta `json:",inline"`
	Roles           []MachineRole `json:"roles,omitempty"` // A list of roles for this Machine to use.
	MachineType     string        `json:"machineType"`
	SSHConfig       SSHConfig     `json:"sshConfig"`
	OS              string        `json:"os"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHClusterProviderConfig struct {
	metav1.TypeMeta `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHMachineProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHClusterProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
}
