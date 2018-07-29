package machine

import "sigs.k8s.io/cluster-api/pkg/kubeadm"

type SSHClientMachineSetupConfigGetter interface {
	GetMachineSetupConfig() (MachineSetupConfig, error)
}

type SSHClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}
