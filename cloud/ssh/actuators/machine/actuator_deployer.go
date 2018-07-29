package machine

// This part of the code implements the machineDeployer Interface used by cluster controller

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"github.com/golang/glog"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
)

// GetIP returns IP of a machine, note that this also REQUIRED by clusterCreator (clusterdeployer.ProviderDeployer)
func (a *Actuator) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	// TODO should we use a dns resolver in case this isnt a valid ip address? i would expect this.
	return machineConfig.SSHConfig.Host, nil
}

// GetKubeConfig returns the kubeconfig file, note that this also REQUIRED by clusterCreator (clusterdeployer.ProviderDeployer)
func (a *Actuator) GetKubeConfig(c *clusterv1.Cluster, m *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	privateKey, passPhrase, err := a.getPrivateKey(c, m)
	if err != nil {
		return "", err
	}

	sshClient := ssh.NewSSHProviderClient(privateKey, passPhrase, machineConfig.SSHConfig)
	return sshClient.GetKubeConfig()
}

func (a *Actuator) getPrivateKey(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, string,  error) {
	machineConfig, err := a.machineProviderConfig(master.Spec.ProviderConfig)
	if err != nil {
		return "", "", err
	}

	secretName := machineConfig.SSHConfig.SecretName
	secret, err := a.kubeClient.CoreV1().Secrets(master.Namespace).Get(secretName, meta_v1.GetOptions{})
	if err != nil {
		glog.Errorf("could not retrieve machine secret", err)
		return "", "", err
	}

	return string(secret.Data["private-key"]), string(secret.Data["pass-phrase"]), nil
}
