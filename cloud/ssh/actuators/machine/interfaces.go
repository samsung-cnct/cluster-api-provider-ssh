package machine

import (
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
)

type SSHClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type SSHClientMachineSetupConfigGetter interface {
	GetMachineSetupConfig() (MachineSetupConfig, error)
}

func getOrNewKubeadm(params ActuatorParams) SSHClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
