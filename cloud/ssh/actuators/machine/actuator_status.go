package machine

import (
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AnnotationKey string

const (
	Name           AnnotationKey = "machine-name"
)

func (a *Actuator) getSSHProviderMachineStatus(c *clusterv1.Cluster, m *clusterv1.Machine) (*v1alpha1.SSHMachineProviderStatus, error) {
	var status v1alpha1.SSHMachineProviderStatus

	providerStatusRawE := m.Status.ProviderStatus
	if providerStatusRawE == nil {
		return nil, nil
	}

	err := a.sshProviderConfigCodec.DecodeProviderStatus(providerStatusRawE, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (a *Actuator) updateSSHProviderMachineStatus(c *clusterv1.Cluster, m *clusterv1.Machine, s v1alpha1.SSHMachineStatus) error {
	ms := v1alpha1.SSHMachineProviderStatus{Status: s}

	err := a.updateAnnotations(c, m)
	if err != nil {
		return err
	}

	rawe, err := a.sshProviderConfigCodec.EncodeProviderStatus(ms.DeepCopyObject())
	if err != nil {
		return err
	}
	m.Status = clusterv1.MachineStatus{LastUpdated: metav1.Now(),
	ProviderStatus:rawe, Versions: &m.Spec.Versions, }

	_, err = a.v1Alpha1Client.Machines(m.Namespace).UpdateStatus(m)
	return err
}


func (a *Actuator) updateAnnotations(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	name := machine.ObjectMeta.Name

	annotations := machine.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[string(Name)] = name
	machine.ObjectMeta.Annotations = annotations

	_, err := a.v1Alpha1Client.Machines(machine.Namespace).Update(machine)
	if err != nil {
		return err
	}
	return err
}

func (a *Actuator) deleteAnnotations(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	annotations := make(map[string]string)
	machine.ObjectMeta.Annotations = annotations

	_, err := a.v1Alpha1Client.Machines(machine.Namespace).Update(machine)
	if err != nil {
		return err
	}
	return err
}