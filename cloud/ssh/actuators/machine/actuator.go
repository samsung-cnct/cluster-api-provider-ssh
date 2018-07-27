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
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	s "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

const (
	ProviderName = "ssh"
)

func init() {
	actuator, err := NewActuator(ActuatorParams{})
	if err != nil {
		glog.Fatalf("Error creating cluster provisioner for %v : %v", ProviderName, err)
	}
	clustercommon.RegisterClusterProvisioner(ProviderName, actuator)
}

// Actuator is responsible for performing machine reconciliation
type Actuator struct {
	clusterClient          client.ClusterInterface
	eventRecorder          record.EventRecorder
	sshClient              s.SSHProviderClientInterface
	sshProviderConfigCodec *v1alpha1.SSHProviderConfigCodec
	kubeClient             *kubernetes.Clientset
	v1Alpha1Client         client.ClusterV1alpha1Interface
	scheme                 *runtime.Scheme
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClusterClient  client.ClusterInterface
	EventRecorder  record.EventRecorder
	SSHClient      s.SSHProviderClientInterface
	KubeClient     *kubernetes.Clientset
	V1Alpha1Client client.ClusterV1alpha1Interface
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
		eventRecorder:          params.EventRecorder,
		sshClient:              params.SSHClient,
		sshProviderConfigCodec: codec,
		scheme:                 scheme,
	}, nil
}

// Create creates a machine and is invoked by the Machine Controller
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

		machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
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

	return nil}

// Delete deletes a machine and is invoked by the Machine Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	glog.Infof("Deleting machine %v for cluster %v.", machine.Name, cluster.Name)
	return fmt.Errorf("TODO: Not yet implemented")
}

// Update updates a machine and is invoked by the Machine Controller
func (a *Actuator) Update(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	glog.Infof("Updating machine %v for cluster %v.", machine.Name, cluster.Name)
	return fmt.Errorf("TODO: Not yet implemented")
}

// Exists test for the existance of a machine and is invoked by the Machine Controller
func (a *Actuator) Exists(c *clusterv1.Cluster, m *clusterv1.Machine) (bool, error) {
	glog.Infof("Checking if machine %v for cluster %v exists.", m.Name, c.Name)
	// Try to use the last saved status locating the machine
	status, err := a.status(m)
	if err != nil {
		return false, err
	}
	// if status is nil, either it doesnt exist or bootstrapping, however in ssh we assume it exists.
	// so some status must be returned.
	return status != nil, nil
}
