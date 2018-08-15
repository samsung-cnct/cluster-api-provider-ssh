package machine

import (
	"errors"

	"github.com/golang/glog"

	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
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
		a.v1Alpha1Client.Machines(machine.Namespace).UpdateStatus(machine)
	}

	if eventAction != noEventAction {
		a.eventRecorder.Eventf(machine, corev1.EventTypeWarning, "Failed"+eventAction, "%v", err.Reason)
	}

	glog.Errorf("Machine error: %v", err.Message)
	return err
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

	metadata := Metadata{
		Items:         metadataMap,
		StartupScript: metadataMap["startup-script"],
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
