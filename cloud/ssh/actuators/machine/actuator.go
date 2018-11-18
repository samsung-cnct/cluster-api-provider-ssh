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

	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
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
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration("valid machineSetupConfigGetter is required"), createEventAction)
	}

	// First get provider config
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration("Cannot unmarshal machine's providerConfig field: %v", err), createEventAction)
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
		glog.Infof("Machine %s for cluster %s exists, skipping.", m.Name, c.Name)
		return nil
	}

	// The doesn't exist case here.
	glog.Infof("Machine %s for cluster %s doesn't exist.", m.Name, c.Name)

	configParams := &MachineParams{
		Roles:    machineConfig.Roles,
		Versions: m.Spec.Versions,
	}

	metadata, err := a.getMetadata(c, m, configParams, machineConfig.SSHConfig)
	if err != nil {
		return err
	}

	glog.Infof("Metadata retrieved: machine %s for cluster %s", m.Name, c.Name)

	// Here we deploy and run the scripts to the node.
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}

	glog.Infof("Running startup script: machine %s for cluster %s...", m.Name, c.Name)

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)

	if err = sshClient.WriteFile(metadata.StartupScript, "/var/log/startupscript.sh"); err != nil {
		glog.Errorf("Error copying startup script: %v", err)
		return err
	}

	if err = sshClient.ProcessCMD("chmod +x /var/log/startupscript.sh && bash /var/log/startupscript.sh"); err != nil {
		glog.Errorf("running startup script error: %v", err)
		return err
	}

	glog.Infof("Annotating machine %s for cluster %s.", m.Name, c.Name)

	// TODO find a way to do this in the cluster controller
	if util.IsMaster(m) {
		err = a.updateClusterObjectEndpoint(c, m)
		if err != nil {
			return err
		}
		// create kubeconfig secret
		err = a.createKubeconfigSecret(c, m, sshClient)
		if err != nil {
			return err
		}
	}

	a.eventRecorder.Eventf(m, corev1.EventTypeNormal, "Created", "Created Machine %v", m.Name)
	return a.updateAnnotations(c, m)
}

// Delete deletes a machine and is invoked by the Machine Controller
func (a *Actuator) Delete(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	glog.Infof("Deleting machine %s for cluster %s.", m.Name, c.Name)

	if a.machineSetupConfigGetter == nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration("valid machineSetupConfigGetter is required"), deleteEventAction)
	}

	// First get provider config
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(m, apierrors.InvalidMachineConfiguration("Cannot unmarshal machine's providerConfig field: %v", err), deleteEventAction)
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
		glog.Infof("Machine %s for cluster %s does not exists (maybe it is still bootstrapping), skipping.", m.Name, c.Name)
		return nil
	}

	// The exists case here.
	glog.Infof("Machine %s for cluster %s exists.", m.Name, c.Name)

	configParams := &MachineParams{
		Roles:    machineConfig.Roles,
		Versions: m.Spec.Versions,
	}

	metadata, err := a.getMetadata(c, m, configParams, machineConfig.SSHConfig)
	if err != nil {
		return err
	}

	glog.Infof("Metadata retrieved: machine %s for cluster %s", m.Name, c.Name)

	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}

	glog.Infof("Running shutdown script: machine %s for cluster %s...", m.Name, c.Name)

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)
	if err = sshClient.ProcessCMD(metadata.ShutdownScript); err != nil {
		glog.Errorf("running shutdown script failed: %v", err)
		return err
	}

	a.eventRecorder.Eventf(m, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", m.Name)
	return nil
}

// Update updates a machine and is invoked by the Machine Controller
func (a *Actuator) Update(c *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	goalMachineName := goalMachine.Name
	glog.Infof("Updating machine %s for cluster %s.", goalMachineName, c.Name)

	// validate the goal machine
	goalConfig, err := a.machineProviderConfig(goalMachine.Spec.ProviderConfig)
	if err != nil {
		return a.handleMachineError(goalMachine, apierrors.InvalidMachineConfiguration("Cannot unmarshal machine's providerConfig field: %v", err), noEventAction)
	}

	glog.Infof("Machine %s for cluster %s; validating goal machine spec.", goalMachineName, c.Name)
	if verr := a.validateMachine(goalMachine, goalConfig); verr != nil {
		return a.handleMachineError(goalMachine, verr, noEventAction)
	}

	// get the current machine that the goal machine is targeting to update
	glog.Infof("Machine %s for cluster %s; Retrieving current machine if it exists.", goalMachineName, c.Name)
	currentMachine, err := util.GetMachineIfExists(a.v1Alpha1Client.Machines(goalMachine.Namespace), goalMachine.ObjectMeta.Name)
	if err != nil {
		return err
	}

	glog.Infof("Machine %s for cluster %s: Retrieving currently installed versions.", goalMachineName, c.Name)
	currentVersionInfo, err := a.getMachineInstanceVersions(c, currentMachine)
	if err != nil {
		return err
	}

	glog.V(3).Infof("Machine versions: %+v", currentVersionInfo)
	currentMachineName := currentMachine.ObjectMeta.Name
	goalVersions := goalMachine.Spec.Versions

	if goalVersions.ControlPlane == currentVersionInfo.ControlPlane && goalVersions.Kubelet == currentVersionInfo.Kubelet {
		glog.Infof("Machine %s for cluster %s: not required.", goalMachineName, c.Name)
		return nil
	}

	currentMachine.Spec.Versions = *currentVersionInfo

	if util.IsMaster(currentMachine) {
		glog.Infof("Performing an in-place upgrade for master %s.", currentMachineName)
		// TODO: should we support custom CAs here?
		if err = a.updateMasterInPlace(c, currentMachine, goalMachine); err != nil {
			glog.Errorf("master in-place update failed for %s: %v", currentMachineName, err)
			return err
		}
	} else {
		glog.Infof("Performing upgrade for worker %s.", currentMachineName)
		if err = a.updateWorkerInPlace(c, currentMachine, goalMachine); err != nil {
			glog.Errorf("worker update failed for %s: v%", currentMachineName, err)
			return err
		}
	}

	a.eventRecorder.Eventf(goalMachine, corev1.EventTypeNormal, "Updated", "Updated Machine %v", goalMachine.Name)
	return a.updateAnnotations(c, goalMachine)
}

// Exists test for the existence of a machine and is invoked by the Machine Controller
func (a *Actuator) Exists(c *clusterv1.Cluster, m *clusterv1.Machine) (bool, error) {
	glog.Infof("Checking if machine %s for cluster %s exists.", m.Name, c.Name)
	// Try to use the last saved status locating the machine
	status, err := a.status(m)
	if err != nil {
		return false, err
	}
	// if status is nil, either it doesn't exist or bootstrapping, however in ssh we assume it exists.
	// so some status must be returned.
	return status != nil, nil
}
