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

package machine

import (
	"fmt"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

const (
	ProviderName = "ssh"
)

func init() {
	sshProviderClient, err := ssh.NewSSHProviderClient()
	if err != nil {
		glog.Fatalf("Error creating ssh provider client: %s", err)
	}

	actuator, err := NewActuator(ActuatorParams{SSHClient: sshProviderClient})
	if err != nil {
		glog.Fatalf("Error creating cluster provisioner for %v : %v", ProviderName, err)
	}
	clustercommon.RegisterClusterProvisioner(ProviderName, actuator)
}

type EventAction string

const (
	createEventAction EventAction = "Create"
	deleteEventAction EventAction = "Delete"
	noEventAction     EventAction = ""
)

// Actuator is responsible for performing machine reconciliation
type Actuator struct {
	clusterClient            client.ClusterInterface
	sshProviderConfigCodec   *v1alpha1.SSHProviderConfigCodec
	kubeadm                  SSHClientKubeadm
	scheme                   *runtime.Scheme
	v1Alpha1Client           client.ClusterV1alpha1Interface
	machineSetupConfigGetter SSHClientMachineSetupConfigGetter
	eventRecorder            record.EventRecorder
	kubeClient               *kubernetes.Clientset
	sshClient                ssh.SSHProviderClientInterface
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClusterClient            client.ClusterInterface
	Kubeadm                  SSHClientKubeadm
	MachineSetupConfigGetter SSHClientMachineSetupConfigGetter
	V1Alpha1Client           client.ClusterV1alpha1Interface
	EventRecorder            record.EventRecorder
	KubeClient               *kubernetes.Clientset
	SSHClient                ssh.SSHProviderClientInterface
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) (*Actuator, error) {

	scheme, err := v1alpha1.NewScheme()
	if err != nil {
		return nil, err
	}
	codec, err := v1alpha1.NewCodec()
	if err != nil {
		return nil, err
	}

	return &Actuator{
		clusterClient:          params.ClusterClient,
		sshProviderConfigCodec: codec,
		kubeadm:                getOrNewKubeadm(params),
		scheme:                 scheme,
		machineSetupConfigGetter: params.MachineSetupConfigGetter,
		v1Alpha1Client:           params.V1Alpha1Client,
		eventRecorder:            params.EventRecorder,
		kubeClient:               params.KubeClient,
	}, nil
}

// Create a machine, its invoked by the Machine Controller
// For ssh provider, we assume machines have been created and exist, if they do not, create
// should result in an error, either way this is a no-op
func (a *Actuator) Create(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Creating m %v for c %v.", m.Name, c.Name)

	exists, err := a.Exists(c, m)
	if err != nil {
		return err
	}

	if exists {
		// update annotations
		if a.v1Alpha1Client != nil {
			return a.updateAnnotations(c, m)
		}

		machineConfig, err := a.machineproviderconfig(m.Spec.ProviderConfig)
		if err != nil {
			return err
		}

		privateKey, err := a.getPrivateKey(c, m)
		if err != nil {
			return err
		}

		a.sshClient.WritePublicKeys(privateKey, machineConfig.SSHConfig)

	} else {
		fmt.Errorf("machine doesnt exist!")
	}

	return nil
}

// Delete a machine, its invoked by the Machine Controller
// Should lead to the deletion of the crd, however not the actual instance.
func (a *Actuator) Delete(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Deleting machine %v for cluster %v.", m.Name, c.Name)
	instance, err := a.machineIfExists(c, m)
	if err != nil {
		return err
	}
	machineConfig, err := a.machineproviderconfig(instance.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	privateKey, err := a.getPrivateKey(c, m)
	if err != nil {
		return err
	}

	a.sshClient.DeletePublicKeys(privateKey, machineConfig.SSHConfig)
	a.eventRecorder.Eventf(m, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", m.ObjectMeta.Name)
	return nil
}

// Update a machine, its invoked by the Machine Controller
func (a *Actuator) Update(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Updating machine %v for cluster %v.", m.Name, c.Name)
	return fmt.Errorf("TODO: Not yet implemented")
}

// Exists test for the existance of a machine and is invoked by the Machine Controller
func (a *Actuator) Exists(c *clusterv1.Cluster, m *clusterv1.Machine) (bool, error) {
	glog.Info("Checking if machine %v for cluster %v exists.", m.Name, c.Name)
	// Try to use the last saved status locating the m
	// in case instance details like the proj or zone has changed
	status, err := a.status(m)
	if err != nil {
		return false, err
	}
	// if status is nil, either it doesnt exist or bootstrapping, however in ssh we assume it exists.
	// so some status must be returned.
	return status != nil, nil
}
