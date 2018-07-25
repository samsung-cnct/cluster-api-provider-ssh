package ssh

import (
	"fmt"

	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
)

type SSHProviderClientInterface interface {
	ProcessCMD(privateKey string, machineSSHConfig v1alpha1.SSHConfig, cmd string) error
	WritePublicKeys(privateKey string, machineSSHConfig v1alpha1.SSHConfig) error
	DeletePublicKeys(privateKey string, machineSSHConfig v1alpha1.SSHConfig) error
	GetKubeConfig(privateKey string, machineSSHConfig v1alpha1.SSHConfig) (string, error)
}

type sshProviderClient struct {
}

func NewSSHProviderClient() (*sshProviderClient, error) {
	return &sshProviderClient{}, nil
}

func (s *sshProviderClient) WritePublicKeys(privateKey string, machineSSHConfig v1alpha1.SSHConfig) error {
	return nil
}

func (s *sshProviderClient) DeletePublicKeys(privateKey string, machineSSHConfig v1alpha1.SSHConfig) error {
	return nil
}

func (s *sshProviderClient) GetKubeConfig(privateKey string, machineSSHConfig v1alpha1.SSHConfig) (string, error) {
	cmd := "sudo cat /etc/kubernetes/admin.conf"
	session, err := GetBasicSession(privateKey, machineSSHConfig)
	if err != nil {
		return "", fmt.Errorf("Failed to create session: %s", err)
	}

	outputBytes, err := session.Output(cmd)

	return string(outputBytes), err

}

func (s *sshProviderClient) ProcessCMD(privateKey string, machineSSHConfig v1alpha1.SSHConfig, cmd string) error {
	session, err := GetBasicSession(privateKey, machineSSHConfig)
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}

	return session.Run(cmd)
}

func GetBasicSession(privateKey string, machineSSHConfig v1alpha1.SSHConfig) (*ssh.Session, error) {
	authMethod, err := PublicKeyFile(privateKey)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: machineSSHConfig.Username,
		Auth: []ssh.AuthMethod{authMethod},
	}
	host := machineSSHConfig.Host
	port := machineSSHConfig.Port

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	return session, nil
}

func PublicKeyFile(privateKey string) (ssh.AuthMethod, error) {
	buffer := []byte(privateKey)

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}
