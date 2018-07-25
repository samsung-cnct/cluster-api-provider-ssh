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

package cluster

import (
	"fmt"

	"github.com/golang/glog"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

// Actuator is responsible for performing cluster reconciliation
type Actuator struct {
	clusterClient          client.ClusterInterface
	sshProviderConfigCodec *v1alpha1.SSHProviderConfigCodec
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClusterClient client.ClusterInterface
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) (*Actuator, error) {

	codec, err := v1alpha1.NewCodec()
	if err != nil {
		return nil, err
	}

	return &Actuator{
		clusterClient:          params.ClusterClient,
		sshProviderConfigCodec: codec,
	}, nil
}

// Reconcile reconciles a cluster and is invoked by the Cluster Controller
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	glog.Infof("Reconciling cluster %v.", cluster.Name)
	return fmt.Errorf("TODO: Not yet implemented")
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	glog.Infof("Deleting cluster %v.", cluster.Name)
	return fmt.Errorf("TODO: Not yet implemented")
}

func (a *Actuator) clusterProviderConfig(providerConfig clusterv1.ProviderConfig) (*v1alpha1.SSHClusterProviderConfig, error) {
	var config v1alpha1.SSHClusterProviderConfig
	err := a.sshProviderConfigCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
