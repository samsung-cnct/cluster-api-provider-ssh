package machine

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
)

const (
	NameAnnotationKey = "hostname"
	BootstrapLabelKey = "boostrap"

	// This file is a yaml that will be used to create the machine-setup configmap on the machine controller.
	// It contains the supported machine configurations along with the startup scripts and OS image paths that correspond to each supported configuration.
	MachineSetupConfigsFilename = "machine_setup_configs.yaml"

	InstanceStatusAnnotationKey = "instance-status"
)

// If the GCEClient has a client for updating Machine objects, this will set
// the appropriate reason/message on the Machine.Status. If not, such as during
// cluster installation, it will operate as a no-op. It also returns the
// original error for convenience, so callers can do "return handleMachineError(...)".
func (a *Actuator) handleMachineError(machine *clusterv1.Machine, err *apierrors.MachineError, eventAction EventAction) error {
	if a.v1Alpha1Client != nil {
		reason := err.Reason
		message := err.Message
		machine.Status.ErrorReason = &reason
		machine.Status.ErrorMessage = &message
		a.v1Alpha1Client.Machines(machine.Namespace).UpdateStatus(machine)
	}

	if eventAction != noEventAction {
		a.eventRecorder.Eventf(machine, corev1.EventTypeWarning, "Failed"+string(eventAction), "%v", err.Reason)
	}

	glog.Errorf("Machine error: %v", err.Message)
	return err
}

func (a *Actuator) validateMachine(machine *clusterv1.Machine, config *v1alpha1.SSHMachineProviderConfig) *apierrors.MachineError {
	if machine.Spec.Versions.Kubelet == "" {
		return apierrors.InvalidMachineConfiguration("spec.versions.kubelet can't be empty")
	}
	return nil
}

func (a *Actuator) getMetadata(c *clusterv1.Cluster, m *clusterv1.Machine, machineParams *MachineParams) (*Metadata, error) {
	var err error
	metadataMap := make(map[string]string)

	if m.Spec.Versions.Kubelet == "" {
		return nil, errors.New("invalid master configuration: missing Machine.Spec.Versions.Kubelet")
	}

	machineSetupConfigs, err := a.machineSetupConfigGetter.GetMachineSetupConfig()
	if err != nil {
		return nil, err
	}

	machineSetupMetadata, err := machineSetupConfigs.GetMetadata(machineParams)
	if err != nil {
		return nil, err
	}

	// TODO: order in which these are appended may be important.
	if isMaster(machineParams.Roles) {
		if m.Spec.Versions.ControlPlane == "" {
			return nil, a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
				"invalid master configuration: missing Machine.Spec.Versions.ControlPlane"), createEventAction)
		}
		masterMap, err := masterMetadata(c, m, &machineSetupMetadata)
		if err != nil {
			return nil, err
		}

		metadataMap = addStringValueMaps(metadataMap, masterMap)
	}

	if isNode(machineParams.Roles) {
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

	if isEtcd(machineParams.Roles) {
		// TODO we assume we will allow for this in the future, for now ignore.
		//if m.Spec.Versions.ControlPlane == "" {
		//	return nil, a.handleMachineError(m, apierrors.InvalidMachineConfiguration(
		//		"invalid master configuration: missing Machine.Spec.Versions.ControlPlane"), createEventAction)
		//}
		//
		//etcdMap, err := etcdMetadata(c, m, &machineSetupMetadata)
		//if err != nil {
		//	return nil, err
		//}
		//
		//metadataMap = addStringValueMaps(metadataMap, etcdMap)
	}

	metadata := Metadata{
		Items: metadataMap,
	}
	return &metadata, nil
}

func (a *Actuator) machineproviderconfig(providerConfig clusterv1.ProviderConfig) (*v1alpha1.SSHMachineProviderConfig, error) {
	var config v1alpha1.SSHMachineProviderConfig
	err := a.sshProviderConfigCodec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *Actuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		Ttl: time.Duration(10) * time.Minute,
	}
	output, err := a.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		glog.Errorf("unable to create token: %v", err)
		return "", err
	}
	return strings.TrimSpace(output), err
}

func (a *Actuator) updateAnnotations(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	name := machine.ObjectMeta.Name

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[NameAnnotationKey] = name
	_, err := a.v1Alpha1Client.Machines(machine.Namespace).Update(machine)
	if err != nil {
		return err
	}
	err = a.updateStatus(machine)
	return err
}

func (a *Actuator) getPrivateKey(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineproviderconfig(master.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	secretName := machineConfig.SSHConfig.SecretName
	secret, err := a.kubeClient.CoreV1().Secrets(master.Spec.Namespace).Get(secretName, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}

	// here we decide what data we get
	// note that this is base64 encoded stilll
	privateKeyBytes, err := base64.StdEncoding.DecodeString(string(secret.Data["private-key"]))
	if err != nil {
		return "", err
	}

	return string(privateKeyBytes), nil
}

func addStringValueMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}

	return m1
}
