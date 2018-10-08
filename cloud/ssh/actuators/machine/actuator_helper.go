package machine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	corev1errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/util"
)

const (
	createEventAction = "Create"
	deleteEventAction = "Delete"
	noEventAction     = ""
	// TODO should this move to the cluster controller?
	apiServerPort          = 443
	upgradeControlPlaneCmd = "sudo curl -o /usr/bin/kubeadm -sSL https://dl.k8s.io/release/v%[1]s/bin/linux/amd64/kubeadm && " +
		"sudo chmod a+rx /usr/bin/kubeadm && " +
		"sudo kubeadm upgrade apply v%[1]s -y"
	getNodeCmd     = "sudo kubectl get no -o go-template='{{range .items}}{{.metadata.name}}:{{.status.nodeInfo.kubeletVersion}}:{{.metadata.annotations.machine}}{{\"\\n\"}}{{end}}' --kubeconfig /etc/kubernetes/admin.conf"
	drainWorkerCmd = "sudo kubectl drain %[1]s --ignore-daemonsets --delete-local-data --force --kubeconfig /etc/kubernetes/admin.conf"
	uncordonCmd    = "sudo kubectl uncordon %[1]s --kubeconfig /etc/kubernetes/admin.conf"
	drainMasterCmd = "sudo kubectl drain %[1]s --ignore-daemonsets --kubeconfig /etc/kubernetes/admin.conf"
)

func (a *Actuator) machineProviderConfig(providerConfig clusterv1.ProviderConfig) (*v1alpha1.SSHMachineProviderConfig, error) {
	var config v1alpha1.SSHMachineProviderConfig
	err := a.sshProviderConfigCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *Actuator) machineProviderStatus(machine *clusterv1.Machine) (*v1alpha1.SSHMachineProviderStatus, error) {
	status := &v1alpha1.SSHMachineProviderStatus{}
	err := a.sshProviderConfigCodec.DecodeProviderStatus(machine.Status.ProviderStatus, status)
	return status, err
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
		StartupScript:  metadataMap[startupScriptKey],
		ShutdownScript: metadataMap[shutdownScriptKey],
		UpgradeScript:  metadataMap[upgradeScriptKey],
	}
	return &metadata, nil
}

func (a *Actuator) semanticVersion(version string) string {
	semVersion := strings.TrimPrefix(version, "v")
	semVersion = strings.TrimSpace(semVersion)
	return semVersion
}

// getNodeForMachine returns node, version, error
func (a *Actuator) getNodeForMachine(c *clusterv1.Cluster, m *clusterv1.Machine) (string, string, error) {
	masterSSHClient, err := a.getMasterSSHClient(c, m)
	if err != nil {
		glog.Error("Error getting master sshClient")
		return "", "", err
	}
	nodeCmd := getNodeCmd + " | grep " + m.Namespace + "/" + m.Name
	glog.Infof("nodeCmd = %s", nodeCmd)
	output, err := masterSSHClient.ProcessCMDWithOutput(nodeCmd)
	if err != nil {
		glog.Errorf("Error getting node: cmd = %s, error = %s", nodeCmd, err)
		return "", "", err
	}
	strs := strings.Split(string(output), ":")
	if len(strs) < 2 {
		return "", "", errors.New("Error getting node name for machine")
	}
	node := strs[0]
	version := a.semanticVersion(strs[1])
	return node, version, nil
}

func (a *Actuator) createKubeconfigSecret(c *clusterv1.Cluster, m *clusterv1.Machine, sshMasterClient ssh.SSHProviderClientInterface) error {
	if a.kubeClient == nil {
		return fmt.Errorf("kubeclient is nil, should not happen")
	}

	glog.Infof("Getting kubeconfig from master, machine %s cluster %s", m.Name, c.Name)
	output, err := sshMasterClient.GetKubeConfigBytes()
	if err != nil {
		glog.Errorf("Error getting kubeconfig from master for machine %s cluster %s error: %v", m.Name, c.Name, err)
		return err
	}

	// create secret
	coreV1Client := a.kubeClient.CoreV1()
	if coreV1Client == nil {
		return errors.New("createKubeconfigSecret, could not initialize coreV1Client")
	}

	glog.Infof("Creating kubeconfig Secret for cluster name: %s namespace: %s", c.Name, c.Namespace)
	data := map[string][]byte{
		"kubeconfig": output,
	}
	secretMeta := metav1.ObjectMeta{
		Name:      c.Name + "-kubeconfig",
		Namespace: c.Namespace,
	}
	_, err = coreV1Client.Secrets(c.Namespace).Create(&corev1.Secret{
		ObjectMeta: secretMeta,
		Type:       corev1.SecretTypeOpaque,
		Data:       data,
	})
	if err != nil {
		if corev1errors.IsAlreadyExists(err) {
			_, err = coreV1Client.Secrets(c.Namespace).Update(&corev1.Secret{
				ObjectMeta: secretMeta,
				Type:       corev1.SecretTypeOpaque,
				Data:       data,
			})
			if err != nil {
				glog.Infof("createKubeconfigSecret Error from Secrets(%s).Update = %s", c.Namespace, err)
				return err
			}
		} else {
			glog.Infof("createKubeconfigSecret Error from Secrets(%s).Create = %s", c.Namespace, err)
		}
	}
	return err
}

func (a *Actuator) getMasterSSHClient(c *clusterv1.Cluster, m *clusterv1.Machine) (ssh.SSHProviderClientInterface, error) {
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return nil, err
	}
	// init master ip address
	masterSSHConfig := v1alpha1.SSHConfig{Username: machineConfig.SSHConfig.Username,
		Host: c.Status.APIEndpoints[0].Host,
		Port: machineConfig.SSHConfig.Port,
	}
	masterSSHClient := ssh.NewSSHProviderClient(privateKey, passPhrase, masterSSHConfig)
	return masterSSHClient, nil
}

func (a *Actuator) getKubeadmTokenOnMaster(c *clusterv1.Cluster, m *clusterv1.Machine) (string, error) {
	if m.ObjectMeta.DeletionTimestamp != nil {
		// No need to create token on a delete.
		return "", nil
	}
	masterSSHClient, err := a.getMasterSSHClient(c, m)
	if err != nil {
		glog.Error("Error getting master sshClient")
		return "", err
	}
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

func (a *Actuator) updateMasterInPlace(c *clusterv1.Cluster, oldMachine *clusterv1.Machine, newMachine *clusterv1.Machine) error {
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

	// Perform kubeadm upgrade on the ControlPlane
	if oldMachine.Spec.Versions.ControlPlane != newMachine.Spec.Versions.ControlPlane {
		glog.Infof("Updating master node %s; controlplane version from %s to %s.", oldMachine.Name, oldMachine.Spec.Versions.ControlPlane, newMachine.Spec.Versions.ControlPlane)
		cmd := fmt.Sprintf(upgradeControlPlaneCmd, newMachine.Spec.Versions.ControlPlane)
		glog.Infof("updateControlPlaneCmd = %s", cmd)

		err := sshClient.ProcessCMD(cmd)
		if err != nil {
			glog.Errorf("Could not perform kubeadm upgrade on ControlPlane: %v", err)
			return err
		}
	}
	// Upgrade ControlPlane packages (kubelet)
	if oldMachine.Spec.Versions.Kubelet != newMachine.Spec.Versions.Kubelet {
		glog.Infof("updating master node %s; kubelet version from %s to %s.", oldMachine.Name, oldMachine.Spec.Versions.Kubelet, newMachine.Spec.Versions.Kubelet)

		configParams := &MachineParams{
			Roles:    machineConfig.Roles,
			Versions: newMachine.Spec.Versions,
		}
		metadata, err := a.getMetadata(c, newMachine, configParams, machineConfig.SSHConfig)
		if err != nil {
			return err
		}
		node, _, err := a.getNodeForMachine(c, newMachine)
		if err != nil {
			return errors.New("updateMasterInPlace Error getting node name for machine")
		}
		drainCmd := fmt.Sprintf(drainMasterCmd, node)
		glog.Infof("drain on master %s with cmd %s", newMachine.Name, drainCmd)
		err = sshClient.ProcessCMD(drainCmd)
		if err != nil {
			glog.Errorf("could not drain master machine %s: %s", newMachine.Name, err)
			return err
		}
		glog.Infof("running upgrade packages script for master %s", newMachine.Name)
		err = sshClient.ProcessCMD(metadata.UpgradeScript)
		if err != nil {
			glog.Errorf("could not upgrade kubelet version: %s-00 on controlPlane %s: %s", newMachine.Spec.Versions.Kubelet, newMachine.Name, err)
			return err
		}
		uncordCmd := fmt.Sprintf(uncordonCmd, node)
		glog.Infof("uncordon on master %s with cmd %s", newMachine.Name, uncordCmd)
		err = sshClient.ProcessCMD(uncordCmd)
		if err != nil {
			glog.Errorf("could not uncordon master machine %s: %s", newMachine.Name, err)
			return err
		}
		glog.Infof("updating master node %s; done.", oldMachine.Name)
	}
	return nil
}

func (a *Actuator) updateWorkerInPlace(c *clusterv1.Cluster, oldMachine *clusterv1.Machine, newMachine *clusterv1.Machine) error {
	glog.Infof("updating worker node %s", oldMachine.Name)
	machineConfig, err := a.machineProviderConfig(newMachine.Spec.ProviderConfig)
	if err != nil {
		return err
	}
	privateKey, passPhrase, err := a.getPrivateKey(c, newMachine.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return err
	}
	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)
	masterSSHClient, err := a.getMasterSSHClient(c, newMachine)
	if err != nil {
		glog.Error("updateWorkerInPlace Error getting master sshClient")
		return err
	}
	// Upgrade Worker packages (kubelet)
	if oldMachine.Spec.Versions.Kubelet != newMachine.Spec.Versions.Kubelet {
		glog.Infof("updating worker node %s; kubelet version from %s to %s.", oldMachine.Name, oldMachine.Spec.Versions.Kubelet, newMachine.Spec.Versions.Kubelet)

		configParams := &MachineParams{
			Roles:    machineConfig.Roles,
			Versions: newMachine.Spec.Versions,
		}
		metadata, err := a.getMetadata(c, newMachine, configParams, machineConfig.SSHConfig)
		if err != nil {
			return err
		}
		node, _, err := a.getNodeForMachine(c, newMachine)
		if err != nil {
			return errors.New("updateWorkerInPlace Error getting node name for machine")
		}
		drainCmd := fmt.Sprintf(drainWorkerCmd, node)
		glog.Infof("drain on worker %s with cmd %s", newMachine.Name, drainCmd)
		err = masterSSHClient.ProcessCMD(drainCmd)
		if err != nil {
			glog.Errorf("failed to drain worker node %s for machine %s: %s", node, newMachine.Name, err)
			return err
		}
		glog.Infof("running upgrade packages script for worker %s", newMachine.Name)
		err = sshClient.ProcessCMD(metadata.UpgradeScript)
		if err != nil {
			glog.Errorf("could not upgrade kubelet version: %s-00 on worker %s: %s", newMachine.Spec.Versions.Kubelet, newMachine.Name, err)
			return err
		}
		uncordCmd := fmt.Sprintf(uncordonCmd, node)
		glog.Infof("uncordon on worker %s with cmd %s", newMachine.Name, uncordCmd)
		err = masterSSHClient.ProcessCMD(uncordCmd)
		if err != nil {
			glog.Errorf("could not uncordon worker machine %s: %s", newMachine.Name, err)
			return err
		}
		glog.Infof("updating worker node %s; done.", oldMachine.Name)
	}
	return nil
}

func (a *Actuator) getMachineInstanceVersions(c *clusterv1.Cluster, m *clusterv1.Machine) (*clusterv1.MachineVersionInfo, error) {
	glog.Infof("retrieving machine versions: machine %s for cluster %s...", m.Name, c.Name)
	// First get provider config

	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("error retrieving machine %s versions: %v", m.Name, err)
	}

	// Here we deploy and run the scripts to the node.
	privateKey, passPhrase, err := a.getPrivateKey(c, m.Namespace, machineConfig.SSHConfig.SecretName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving machine %s versions: %v", m.Name, err)
	}

	// Get Kubelet version
	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)

	_, version, err := a.getNodeForMachine(c, m)
	if err != nil {
		return nil, errors.New("getMachineInstanceVersions error getting version for machine " + m.Name)
	}
	kubeletVersion := a.semanticVersion(version)

	if util.IsMaster(m) {
		// Get ControlPlane
		// TODO this is way too provisioning specific.
		cmd := "kubeadm version | awk -F, '{print $3}' | awk -F\\\" '{print $2}'"
		kubeadmVersionResult, err := sshClient.ProcessCMDWithOutput(cmd)
		if err != nil {
			glog.Errorf("error, retrieving machine controlPlane versions for machine %s %s: %v", m.Name, cmd, err)
			return nil, err
		}
		controlPlaneVersion := a.semanticVersion(string(kubeadmVersionResult))

		return &clusterv1.MachineVersionInfo{Kubelet: kubeletVersion, ControlPlane: controlPlaneVersion}, nil
	} else {
		return &clusterv1.MachineVersionInfo{Kubelet: kubeletVersion}, nil
	}

}
