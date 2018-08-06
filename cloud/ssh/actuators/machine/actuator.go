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
	"github.com/golang/glog"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/cluster-api/pkg/util"
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
	clusterClient            client.ClusterInterface
	eventRecorder            record.EventRecorder
	sshProviderConfigCodec   *v1alpha1.SSHProviderConfigCodec
	kubeClient               *kubernetes.Clientset
	v1Alpha1Client           client.ClusterV1alpha1Interface
	scheme                   *runtime.Scheme
	machineSetupConfigGetter SSHClientMachineSetupConfigGetter
	kubeadm                  SSHClientKubeadm
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClusterClient            client.ClusterInterface
	EventRecorder            record.EventRecorder
	KubeClient               *kubernetes.Clientset
	V1Alpha1Client           client.ClusterV1alpha1Interface
	MachineSetupConfigGetter SSHClientMachineSetupConfigGetter
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
		sshProviderConfigCodec: codec,
		kubeClient:             params.KubeClient,
		v1Alpha1Client:         params.V1Alpha1Client,
		scheme:                 scheme,
		machineSetupConfigGetter: params.MachineSetupConfigGetter,
		kubeadm:                  kubeadm.New(),
	}, nil
}

// Create creates a machine and is invoked by the Machine Controller
func (a *Actuator) Create(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Creating machine %s for cluster %s.", m.Name, c.Name)
	if a.machineSetupConfigGetter == nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
			"valid machineSetupConfigGetter is required"), createEventAction)
	}

	// First get provider config
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
			"Cannot unmarshal machine's providerConfig field: %v", err), createEventAction)
	}

	// Now validate
	if err := a.validateMachine(m, machineConfig); err != nil {
		return a.handleMachineError(m, err, createEventAction)
	}

	// check if the machine exists (here we mean we haven't provisioned it yet.)
	exists, err := a.Exists(c, m)
	if err != nil {
		return err
	}

	if exists {
		glog.Infof("machine %s for cluster %s exists,skipping creation.", m.Name, c.Name)
		return nil
	}

	// The doesn't exist case here.
	glog.Infof("machine %s for cluster %s doesnt exist; Creating.", m.Name, c.Name)

	configParams := &MachineParams{
		Roles:    machineConfig.Roles,
		Versions: m.Spec.Versions,
	}

	metadata, err := a.getMetadata(c, m, configParams)
	if err != nil {
		return err
	}

	glog.Infof("metadata retrieved: machine %s for cluster %s", m.Name, c.Name)

	// Here we deploy and run the scripts to the node.
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}

	glog.Infof("running startup script: machine %s for cluster %s...", m.Name, c.Name)

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)
	err = sshClient.ProcessCMD(metadata.StartupScript)
	if err != nil {
		glog.Errorf("running startup script error:", err)
		return err
	}

	glog.Infof("updating SSHProviderMachineStatus for machine %s in cluster %s.", m.Name, c.Name)

	a.updateSSHProviderMachineStatus(c, m, v1alpha1.MachineCreated)
	a.eventRecorder.Eventf(m, corev1.EventTypeNormal, "Created", "Created Machine %v", m.Name)
	return nil
}

// Delete deletes a machine and is invoked by the Machine Controller
func (a *Actuator) Delete(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Deleting machine %v for cluster %v.", m.Name, c.Name)

	if a.machineSetupConfigGetter == nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
			"valid machineSetupConfigGetter is required"), deleteEventAction)
	}

	// First get provider config
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
			"Cannot unmarshal machine's providerConfig field: %v", err), deleteEventAction)
	}

	// Now validate
	if err := a.validateMachine(m, machineConfig); err != nil {
		return a.handleMachineError(m, err, deleteEventAction)
	}

	// Check if the machine exists (here we mean it is not bootstrapping.)
	exists, err := a.Exists(c, m)
	if err != nil {
		return err
	}

	if !exists {
		glog.Infof("machine %s for cluster %s does not exists (maybe it is still bootstrapping), skipping deletion.", m.Name, c.Name)
		return nil
	}

	// The exists case here.
	glog.Infof("machine %s for cluster %s exists; Deleting.", m.Name, c.Name)

	configParams := &MachineParams{
		Roles:    machineConfig.Roles,
		Versions: m.Spec.Versions,
	}

	metadata, err := a.getMetadata(c, m, configParams)
	if err != nil {
		return err
	}

	glog.Infof("metadata retrieved: machine %s for cluster %s", m.Name, c.Name)

	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}

	glog.Infof("running shutdown script: machine %s for cluster %s...", m.Name, c.Name)

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)

	glog.Infof("running shutdown script: %s", metadata.ShutdownScript)

	err = sshClient.ProcessCMD(metadata.ShutdownScript)
	if err != nil {
		glog.Errorf("error running shutdown script:", err)
		return err
	}

	// If we have a v1Alpha1Client, then delete the annotations on the machine.
	if a.v1Alpha1Client != nil {
		return a.deleteAnnotations(c, m)
	}

	a.eventRecorder.Eventf(m, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", m.Name)

	return nil
}

// Update updates a machine and is invoked by the Machine Controller
func (a *Actuator) Update(cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	goalMachineName := goalMachine.ObjectMeta.Name
	clusterName := cluster.ObjectMeta.Name
	glog.Infof("Updating Machine %v for cluster %v.", goalMachineName, clusterName)

	// validate the goal machine
	goalConfig, err := a.machineProviderConfig(goalMachine.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(goalMachine, apierrors.InvalidMachineConfiguration("Cannot unmarshal machine's providerConfig field: %v", err), noEventAction)
	}

	glog.Infof("Updating Machine %v for cluster %v: Validating goal machine spec.", goalMachineName, clusterName)
	if verr := a.validateMachine(goalMachine, goalConfig); verr != nil {
		return a.handleMachineError(goalMachine, verr, noEventAction)
	}

	// get the current machine that the goal machine is targeting to update
	glog.Infof("Updating Machine %v for cluster %v: Retrieve current machine if it exists.", goalMachineName, clusterName)
	currentMachine, err := util.GetMachineIfExists(a.v1Alpha1Client.Machines(goalMachine.Namespace), goalMachine.ObjectMeta.Name)
	if err != nil {
		return err
	}

	glog.Infof("Updating Machine %v for cluster %v: Retrieving currently installed versions.", goalMachineName, clusterName)
	currentVersionInfo, err := a.getMachineInstanceVersions(cluster, currentMachine)
	if err != nil {
		return err
	}

	glog.V(3).Infof("machine versions: %+v", currentVersionInfo)
	currentMachineName := currentMachine.ObjectMeta.Name
	goalVersions := goalMachine.Spec.Versions

	if goalVersions.ControlPlane == currentVersionInfo.ControlPlane && goalVersions.Kubelet == currentVersionInfo.Kubelet {
		glog.Infof("No updating required for Machine %s of cluster %s: ", goalMachineName, clusterName)
		return nil
	}

	currentMachine.Spec.Versions = *currentVersionInfo

	if util.IsMaster(currentMachine) {
		glog.Infof("Doing an in-place upgrade for master %s.", currentMachineName)
		// TODO: should we support custom CAs here?
		err = a.updateMasterInplace(cluster, currentMachine, goalMachine)
		if err != nil {
			glog.Errorf("master in-place update failed for %s: %v", currentMachineName, err)
		}
	} else {
		glog.Infof("re-creating machine %s for update. ", currentMachineName)
		err = a.Delete(cluster, currentMachine)
		if err != nil {
			glog.Errorf("delete machine %s for update failed: %v", currentMachineName, err)
		} else {
			err = a.Create(cluster, goalMachine)
			if err != nil {
				glog.Errorf("create machine %s for update failed: %v", goalMachineName, err)
			}
		}
	}
	if err != nil {
		return err
	}

	a.eventRecorder.Eventf(goalMachine, corev1.EventTypeNormal, "Updated", "Updated Machine %v", goalMachine.Name)

	// If we have a v1Alpha1Client, then annotate the machine.
	if a.v1Alpha1Client != nil {
		return a.updateAnnotations(cluster, goalMachine)
	}

	return a.updateStatus(goalMachine)
}

// Exists test for the existance of a machine and is invoked by the Machine Controller
func (a *Actuator) Exists(c *clusterv1.Cluster, m *clusterv1.Machine) (bool, error) {
	glog.Infof("Checking if machine %v for cluster %v exists.", m.Name, c.Name)

	// first check is to get the machine from the api to verify that the machine resource object still exists
	if a.v1Alpha1Client == nil {
		return false, nil
	}
	currentMachine, err := util.GetMachineIfExists(a.v1Alpha1Client.Machines(m.Namespace), m.ObjectMeta.Name)
	if err != nil {
		return false, err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted (or does not exist yet ie. bootstrapping)
		return false, nil
	}

	// now we verify whether we have an sshProviderStatus at all
	status, err := a.getSSHProviderMachineStatus(c, currentMachine)
	if err != nil {
		return false, err
	}

	glog.Infof("Machine %s for cluster %s exists? %t", m.Name, c.Name, status != nil)
	return status != nil, nil
}
