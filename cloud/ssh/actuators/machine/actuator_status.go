package machine

import (
	"fmt"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)



const (
	Name = "machine-name"
)

func (a *Actuator) updateStatusAndAnnotations(c *clusterv1.Cluster, m *clusterv1.Machine, s v1alpha1.SSHMachineStatus) error {
	err := a.updateStatus(c, m, s)
	if err != nil {
		return err
	}

	return a.updateAnnotations(c, m)
}

// Sets the status of the instance identified by the given machine to the given machine
func (a *Actuator) updateStatus(c *clusterv1.Cluster, m *clusterv1.Machine, s v1alpha1.SSHMachineStatus) error {
	if a.v1Alpha1Client == nil {
		return nil
	}

	exists, err := a.Exists(c, m)
	if err != nil {
		return err
	}

	if !exists {
		// The current status no longer exists because the matching CRD has been deleted.
		return fmt.Errorf("Machine has already been deleted. Cannot update current instance status for machine %s", m.Name)
	}

	m.Status = clusterv1.MachineStatus{
		LastUpdated: metav1.Now(),
		Versions: &m.Spec.Versions,
	}

	_, err = a.v1Alpha1Client.Machines(m.Namespace).UpdateStatus(m)
	return err
}

func (a *Actuator) updateAnnotations(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	if a.v1Alpha1Client == nil {
		return nil
	}

	annotations := m.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[Name] = m.Name
	m.ObjectMeta.Annotations = annotations

	_, err := a.v1Alpha1Client.Machines(m.Namespace).Update(m)
	return err
}

func (a *Actuator) deleteAnnotations(c *clusterv1.Cluster, m *clusterv1.Machine) error {
	if a.v1Alpha1Client == nil {
		return nil
	}

	annotations := make(map[string]string)
	m.ObjectMeta.Annotations = annotations

	_, err := a.v1Alpha1Client.Machines(m.Namespace).Update(m)
	return err
}
