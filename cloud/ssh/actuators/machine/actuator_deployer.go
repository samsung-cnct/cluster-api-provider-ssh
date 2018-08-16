package machine

// This part of the code implements the machineDeployer Interface used by cluster controller

import (
	"fmt"

	"github.com/golang/glog"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// GetKubeConfig returns the kubeconfig file, note that this also REQUIRED by clusterCreator (clusterdeployer.ProviderDeployer)
func (a *Actuator) GetKubeConfig(c *clusterv1.Cluster, m *clusterv1.Machine) (string, error) {
	machineConfig, err := a.machineProviderConfig(m.Spec.ProviderConfig)
	if err != nil {
		return "", err
	}

	// this is used primarily by clusterctl which is run in the machine that starts up a call to the external cluster
	// as such, this does not actually do much with the machine exect execute a command
	sshClient := ssh.NewSSHProviderClient("", "", machineConfig.SSHConfig)
	return sshClient.GetKubeConfig()
}

func (a *Actuator) getPrivateKey(c *clusterv1.Cluster, namespace string, secretName string) (string, string, error) {

	if a.kubeClient == nil {
		return "", "", fmt.Errorf("kubeclient is nil, should not happen")
	}

	coreV1Client := a.kubeClient.CoreV1()

	if coreV1Client != nil {
		glog.Infof("machine info: %s, %s", namespace, secretName)
		secretsClient := coreV1Client.Secrets(namespace)

		secret, err := secretsClient.Get(secretName, meta_v1.GetOptions{})
		if err != nil {
			glog.Errorf("could not retrieve machine secret", err)
			return "", "", err
		}

		return string(secret.Data["private-key"]), string(secret.Data["pass-phrase"]), nil
	}

	return "", "", fmt.Errorf("core v1 client is nil, should not happen")
}
