package machine

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	createEventAction = "Create"
	deleteEventAction = "Delete"
	noEventAction     = ""
)

func (a *Actuator) machineProviderConfig(providerConfig clusterv1.ProviderConfig) (*v1alpha1.SSHMachineProviderConfig, error) {
	var config v1alpha1.SSHMachineProviderConfig
	err := a.sshProviderConfigCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *Actuator) validateMachine(machine *clusterv1.Machine, config *v1alpha1.SSHMachineProviderConfig) *apierrors.MachineError {
	if machine.Spec.Versions.Kubelet == "" {
		return apierrors.InvalidMachineConfiguration("spec.versions.kubelet can't be empty")
	}
	return nil
}

func (a *Actuator) handleMachineError(machine *clusterv1.Machine, err *apierrors.MachineError, eventAction string) error {
	if a.v1Alpha1Client != nil {
		reason := err.Reason
		message := err.Message
		machine.Status.ErrorReason = &reason
		machine.Status.ErrorMessage = &message
		//nolint:errcheck
		a.v1Alpha1Client.Machines(machine.Namespace).UpdateStatus(machine)
	}

	if eventAction != noEventAction {
		a.eventRecorder.Eventf(machine, corev1.EventTypeWarning, "Failed"+eventAction, "%v", err.Reason)
	}

	glog.Errorf("Machine error: %v", err.Message)
	return err
}

func (a *Actuator) getMetadata(c *clusterv1.Cluster, m *clusterv1.Machine, machineParams *MachineParams, sshConfig v1alpha1.SSHConfig) (*Metadata, error) {
	var err error
	metadataMap := make(map[string]string)

	if m.Spec.Versions.Kubelet == "" {
		return nil, errors.New("invalid configuration: missing Machine.Spec.Versions.Kubelet")
	}

	machineSetupConfigs, err := a.machineSetupConfigGetter.GetMachineSetupConfig()
	if err != nil {
		return nil, err
	}

	machineSetupMetadata, err := machineSetupConfigs.GetMetadata(machineParams)
	if err != nil {
		return nil, err
	}

	if util.IsMaster(m) {
		if m.Spec.Versions.ControlPlane == "" {
			return nil, a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
				"invalid master configuration: missing Machine.Spec.Versions.ControlPlane"), createEventAction)
		}

		masterMap, err := masterMetadata(c, m, &machineSetupMetadata, sshConfig)
		if err != nil {
			return nil, err
		}

		metadataMap = addStringValueMaps(metadataMap, masterMap)
	} else {
		kubeadmToken, err := a.getKubeadmToken()
		if err != nil {
			return nil, err
		}

		nodeMap, err := nodeMetadata(kubeadmToken, c, m, &machineSetupMetadata)
		if err != nil {
			return nil, err
		}

		metadataMap = addStringValueMaps(metadataMap, nodeMap)
	}

	metadata := Metadata{
		Items:          metadataMap,
		StartupScript:  metadataMap["startup-script"],
		ShutdownScript: metadataMap["shutdown-script"],
	}
	return &metadata, nil
}

func (a *Actuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		Ttl: time.Duration(10) * time.Minute,
	}
	output, err := a.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		glog.Errorf("unable to create token: %s, %v", output, err)
		return "", err
	}
	return strings.TrimSpace(output), err
}

func addStringValueMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}

	return m1
}

func (a *Actuator) updateMasterInplace(c *clusterv1.Cluster, oldMachine *clusterv1.Machine, newMachine *clusterv1.Machine) error {
	glog.Infof("updating master node %s", oldMachine.Name)
	machineConfig, err := a.machineProviderConfig(newMachine.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	privateKey, passPhrase, err := a.getPrivateKey(c, newMachine.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)

	// Upgrade ControlPlane items
	if oldMachine.Spec.Versions.ControlPlane != newMachine.Spec.Versions.ControlPlane {
		glog.Infof("updating master node %s; controlplane version from %s to %s", oldMachine.Name, oldMachine.Spec.Versions.ControlPlane, newMachine.Spec.Versions.ControlPlane)

		cmd := fmt.Sprintf(
			"curl -sSL https://dl.k8s.io/release/v%s/bin/linux/amd64/kubeadm | sudo tee /usr/bin/kubeadm > /dev/null; "+
				"sudo chmod a+rx /usr/bin/kubeadm", newMachine.Spec.Versions.ControlPlane)
		err := sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("could not install kubeadm binary: %v", err)
			return err
		}

		// Upgrade control plane.
		cmd = fmt.Sprintf("sudo kubeadm upgrade apply %s -y", "v"+newMachine.Spec.Versions.ControlPlane)
		err = sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("could not upgrade to new version %s: %v", newMachine.Spec.Versions.ControlPlane, err)
			return err
		}
	}

	// Upgrade Kubelet.
	if oldMachine.Spec.Versions.Kubelet != newMachine.Spec.Versions.Kubelet {
		glog.Infof("updating master node %s; kubelet version from %s to %s.", oldMachine.Name, oldMachine.Spec.Versions.Kubelet, newMachine.Spec.Versions.Kubelet)
		cmd := fmt.Sprintf("sudo kubectl drain %s --kubeconfig /etc/kubernetes/admin.conf --ignore-daemonsets", newMachine.Name)
		// The errors are intentionally ignored as master has static pods.
		_ = sshClient.ProcessCMD(cmd)

		// Upgrade kubelet to desired version.
		cmd = fmt.Sprintf("sudo apt install kubelet=%s-00", newMachine.Spec.Versions.Kubelet)
		err = sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("Could not apt install Kubelet version: %s-00", newMachine.Spec.Versions.Kubelet+"-00: %v", err)
			return err
		}

		cmd = fmt.Sprintf("sudo kubectl uncordon %s --kubeconfig /etc/kubernetes/admin.conf", newMachine.Name)
		err = sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("Could not uncordon the node: %s: %v", newMachine.Name, err)
			return err
		}
	}

	glog.Infof("updating master node %s; done.", oldMachine.Name)

	return nil
}

func (a *Actuator) getMachineInstanceVersions(c *clusterv1.Cluster, m *clusterv1.Machine) (*clusterv1.MachineVersionInfo, error) {
	glog.Infof("retrieving machine versions: machine %s for cluster %s...", m.Name, c.Name)
	// First get provider config
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("retrieving machine versions: %v", err)
	}

	// Here we deploy and run the scripts to the node.
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return nil, fmt.Errorf("retrieving machine versions: %v", err)
	}

	// Get Kubelet version
	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)

	versionResult, err := sshClient.ProcessCMDWithOutput("kubelet --version")
	if err != nil {
		glog.Errorf("retrieving machine versions: %v", err)
		return nil, err
	}

	kubeletVersion := strings.Split(string(versionResult), " v")[1]
	kubeletVersion = strings.TrimSpace(kubeletVersion)

	if util.IsMaster(m) {
		// Get ControlPlane
		// TODO this is way too provisioning specific.
		cmd := "kubeadm version | awk -F, '{print $3}' | awk -F\\\" '{print $2}'"
		versionResult, err = sshClient.ProcessCMDWithOutput(cmd)
		if err != nil {
			glog.Errorf("error, retrieving machine controlPlane versions %s: %v", cmd, err)
			return nil, err
		}

		controlPlaneVersion := strings.Replace(string(versionResult), "v", "", 1)
		controlPlaneVersion = strings.TrimSpace(controlPlaneVersion)

		return &clusterv1.MachineVersionInfo{Kubelet: kubeletVersion, ControlPlane: controlPlaneVersion}, nil
	} else {
		return &clusterv1.MachineVersionInfo{Kubelet: kubeletVersion}, nil
	}

}
