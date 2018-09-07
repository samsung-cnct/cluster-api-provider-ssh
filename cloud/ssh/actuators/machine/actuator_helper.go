package machine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	createEventAction = "Create"
	deleteEventAction = "Delete"
	noEventAction     = ""
	// TODO should this move to the cluster controller?
	apiServerPort = 443
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

	glog.Errorf("machine %s error: %v", machine.Name, err.Message)
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
		// create token on the master
		if len(c.Status.APIEndpoints) < 1 {
			msg := fmt.Sprintf("getMetadata: The master APIEndpoint has not been initialized, machine %s cluster %s", m.Name, c.Name)
			return nil, errors.New(msg)
		}
		kubeadmToken, err := a.getKubeadmTokenOnMaster(c, m)
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

func (a *Actuator) getKubeadmTokenOnMaster(c *clusterv1.Cluster, m *clusterv1.Machine) (string, error) {
	if m.ObjectMeta.DeletionTimestamp != nil {
		// No need to create token on a delete.
		return "", nil
	}
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return "", err
	}
	// init master ip address
	masterSSHConfig := v1alpha1.SSHConfig{Username: machineConfig.SSHConfig.Username,
		Host: c.Status.APIEndpoints[0].Host,
		Port: machineConfig.SSHConfig.Port,
	}

	masterSSHClient := ssh.NewSSHProviderClient(privateKey, passPhrase, masterSSHConfig)

	glog.Infof("Creating token on master, machine %s cluster %s", m.Name, c.Name)
	// TODO use kubeadm ttl option and try without full path
	output, err := masterSSHClient.ProcessCMDWithOutput("sudo /usr/bin/kubeadm token create")
	if err != nil {
		glog.Errorf("Error creating token on master for machine %s cluster %s error: %v", m.Name, c.Name, err)
		return "", err
	}
	token := string(output)
	glog.Infof("Token created successfully for machine %s cluster %s", m.Name, c.Name)
	return strings.TrimSpace(token), err
}

func addStringValueMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}

	return m1
}

// TODO move this to the cluster controller?
func (a *Actuator) updateClusterObjectEndpoint(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	if len(c.Status.APIEndpoints) == 0 {
		masterIP, err := a.GetIP(c, m)
		if err != nil {
			return err
		}
		c.Status.APIEndpoints = append(c.Status.APIEndpoints,
			clusterv1.APIEndpoint{
				Host: masterIP,
				Port: apiServerPort,
			})

		_, err = a.v1Alpha1Client.Clusters(c.Namespace).UpdateStatus(c)
		return err
	}
	return nil
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
		glog.Infof("Updating master node %s; controlplane version from %s to %s.", oldMachine.Name, oldMachine.Spec.Versions.ControlPlane, newMachine.Spec.Versions.ControlPlane)

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
			glog.Errorf("failed to upgrade to new version %s: %v", newMachine.Spec.Versions.ControlPlane, err)
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
			glog.Errorf("could not apt install Kubelet version: %s-00", newMachine.Spec.Versions.Kubelet+"-00: %v", err)
			return err
		}

		cmd = fmt.Sprintf("sudo kubectl uncordon %s --kubeconfig /etc/kubernetes/admin.conf", newMachine.Name)
		err = sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("failed to uncordon the node: %s: %v", newMachine.Name, err)
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
