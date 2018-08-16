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
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	providerconfigv1 "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

// Actuator is responsible for performing cluster reconciliation
type Actuator struct {
	clusterClient          client.ClusterInterface
	v1Alpha1Client         client.ClusterV1alpha1Interface
	sshProviderConfigCodec *providerconfigv1.SSHProviderConfigCodec
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClusterClient  client.ClusterInterface
	V1Alpha1Client client.ClusterV1alpha1Interface
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) (*Actuator, error) {
	codec, err := providerconfigv1.NewCodec()
	if err != nil {
		return nil, err
	}

	return &Actuator{
		clusterClient:          params.ClusterClient,
		v1Alpha1Client:         params.V1Alpha1Client,
		sshProviderConfigCodec: codec,
	}, nil
}

func (a *Actuator) machineProviderConfig(providerConfig clusterv1.ProviderConfig) (*providerconfigv1.SSHMachineProviderConfig, error) {
	var config providerconfigv1.SSHMachineProviderConfig
	err := a.sshProviderConfigCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Reconcile reconciles a cluster and is invoked by the Cluster Controller
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	glog.Infof("Reconciling cluster %v.", cluster.Name)

	machineList, err := a.v1Alpha1Client.Machines(cluster.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var newAPIEndpoints []clusterv1.APIEndpoint
	for _, m := range machineList.Items {
		if util.IsMaster(&m) {
			config, err := a.machineProviderConfig(m.Spec.ProviderConfig)
			if err != nil {
				return err
			}

			newAPIEndpoints = append(newAPIEndpoints,
				clusterv1.APIEndpoint{Host: config.SSHConfig.Host,
					Port: config.SSHConfig.Port})
		}
	}
	cluster.Status.APIEndpoints = newAPIEndpoints

	_, err = a.v1Alpha1Client.Clusters(cluster.Namespace).UpdateStatus(cluster)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	glog.Infof("Deleting cluster %v.", cluster.Name)

	// The core machine controller will not [delete](https://goo.gl/LEW9s1)
	// a machine unless it [has a cluster](https://goo.gl/X8AGH6). Therefore
	// we must assume one cluster per namespace. Related issues:
	//
	// https://github.com/samsung-cnct/cluster-api-provider-ssh/pull/50
	// https://github.com/kubernetes-sigs/cluster-api/issues/252
	// https://github.com/kubernetes-sigs/cluster-api/issues/177
	// https://github.com/kubernetes-sigs/cluster-api/issues/41
	return nil
}
