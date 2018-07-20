package machine

// This part of the code implements the machineDeployer Interface used by cluster controller

import (
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// GetIP returns IP of a machine, note that this also REQUIRED by clusterCreator (clusterdeployer.ProviderDeployer)
func (a *Actuator) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineproviderconfig(machine.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	// TODO should we use a dns resolver in case this isnt a valid ip address? i would expect this.
	return machineConfig.SSHConfig.Host, nil
}

// GetKubeConfig returns the kubeconfig file, note that this also REQUIRED by clusterCreator (clusterdeployer.ProviderDeployer)
func (a *Actuator) GetKubeConfig(c *clusterv1.Cluster, m *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineproviderconfig(m.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	privateKey, err := a.getPrivateKey(c, m)
	if err != nil {
		return "", err
	}

	return a.sshClient.GetKubeConfig(privateKey, machineConfig.SSHConfig)
}
