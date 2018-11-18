package machine

import (
	"fmt"

	"bytes"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

type MachineStatus *clusterv1.Machine

type AnnotationKey string

const (
	InstanceStatus AnnotationKey = "instance-status"
	Name           AnnotationKey = "machine-name"
)

// Get the status of the instance identified by the given machine
func (a *Actuator) status(m *clusterv1.Machine) (MachineStatus, error) {
	if a.v1Alpha1Client == nil {
		return nil, nil
	}
	currentMachine, err := util.GetMachineIfExists(a.v1Alpha1Client.Machines(m.Namespace), m.ObjectMeta.Name)
	if err != nil {
		return nil, err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted (or does not exist yet ie. bootstrapping)
		return nil, nil
	}
	return a.machineStatus(currentMachine)
}

// Gets the state of the instance stored on the given machine CRD
func (a *Actuator) machineStatus(m *clusterv1.Machine) (MachineStatus, error) {
	if m.ObjectMeta.Annotations == nil {
		return nil, nil
	}

	annot := m.ObjectMeta.Annotations[string(InstanceStatus)]
	if annot == "" {
		return nil, nil
	}

	serializer := json.NewSerializer(json.DefaultMetaFactory, a.scheme, a.scheme, false)
	var status clusterv1.Machine
	_, _, err := serializer.Decode([]byte(annot), &schema.GroupVersionKind{Group: "", Version: "cluster.k8s.io/v1alpha1", Kind: "Machine"}, &status)
	if err != nil {
		return nil, fmt.Errorf("decoding failure: %v", err)
	}

	return MachineStatus(&status), nil
}

// Sets the status of the instance identified by the given machine to the given machine
func (a *Actuator) updateStatus(machine *clusterv1.Machine) error {
	if a.v1Alpha1Client == nil {
		return nil
	}
	status := MachineStatus(machine)
	currentMachine, err := util.GetMachineIfExists(a.v1Alpha1Client.Machines(machine.Namespace), machine.ObjectMeta.Name)
	if err != nil {
		return err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted.
		return fmt.Errorf("Machine has already been deleted. Cannot update current instance status for machine %v", machine.ObjectMeta.Name)
	}

	m, err := a.setMachineStatus(currentMachine, status)
	if err != nil {
		return err
	}

	_, err = a.v1Alpha1Client.Machines(machine.Namespace).Update(m)
	return err
}

// Applies the state of an instance onto a given machine CRD
func (a *Actuator) setMachineStatus(machine *clusterv1.Machine, status MachineStatus) (*clusterv1.Machine, error) {
	// Avoid status within status within status ...
	status.ObjectMeta.Annotations[string(InstanceStatus)] = ""

	serializer := json.NewSerializer(json.DefaultMetaFactory, a.scheme, a.scheme, false)
	b := []byte{}
	buff := bytes.NewBuffer(b)
	err := serializer.Encode((*clusterv1.Machine)(status), buff)
	if err != nil {
		return nil, fmt.Errorf("encoding failure: %v", err)
	}

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[string(InstanceStatus)] = buff.String()
	return machine, nil
}

func (a *Actuator) updateAnnotations(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	if a.v1Alpha1Client == nil {
		return nil
	}

	name := machine.ObjectMeta.Name

	annotations := machine.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[string(Name)] = name
	machine.ObjectMeta.Annotations = annotations

	return a.updateStatus(machine)
}
