package ssh

import (
	"fmt"
	"net"
	"os"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
)

type SSHProviderClientInterface interface {
	ProcessCMD(cmd string) error
	WritePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error
	DeletePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error
	GetKubeConfig() (string, error)
}

type sshProviderClient struct {
	username   string
	address    string
	port       int
	privateKey string
	passPhrase string
}

func NewSSHProviderClient(privateKey string, passPhrase string, machineSSHConfig v1alpha1.SSHConfig) *sshProviderClient {
	return &sshProviderClient{
		username:   machineSSHConfig.Username,
		address:    machineSSHConfig.Host,
		port:       machineSSHConfig.Port,
		privateKey: privateKey,
		passPhrase: passPhrase,
	}
}

func (s *sshProviderClient) WritePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error {
	return nil
}

func (s *sshProviderClient) DeletePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error {
	return nil
}

func (s *sshProviderClient) GetKubeConfig() (string, error) {
	cmd := "sudo cat /etc/kubernetes/admin.conf"

	session, err := GetBasicSession(s)
	if err != nil {
		return "", fmt.Errorf("Failed to create session: %s", err)
	}
	defer session.Close()

	outputBytes, err := session.Output(cmd)

	return string(outputBytes), err
}

func (s *sshProviderClient) ProcessCMD(cmd string) error {
	session, err := GetBasicSession(s)
	if err != nil {
		return fmt.Errorf("Failed to create session: %v", err)
	}

	defer session.Close()

	return session.Run(cmd)
}

func GetBasicSession(s *sshProviderClient) (*ssh.Session, error) {
	var sshConfig *ssh.ClientConfig
	sshAuthMethods := make([]ssh.AuthMethod, 0)

	if s.privateKey != "" {
		publicKeyMethod, err := PublicKeyFile(s.privateKey, s.passPhrase)
		if err != nil {
			return nil, err
		}
		sshAuthMethods = append(sshAuthMethods, publicKeyMethod)
	}

	sshAgent := SSHAgent()
	if sshAgent != nil {
		sshAuthMethods = append(sshAuthMethods, sshAgent)
	}

	sshConfig = &ssh.ClientConfig{
		User: s.username,
		Auth: sshAuthMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.address, s.port), sshConfig)
	if err != nil {
		emsg := fmt.Sprintf("failed to dial to %s:%d:", s.address, s.port)
		return nil, fmt.Errorf(emsg, err)
	}

	session, err := connection.NewSession()
	if err != nil {
		glog.Errorf("failed to create sesssion", err)
		return nil, fmt.Errorf("Failed to create session: %v", err)
	}

	return session, nil
}

func PublicKeyFile(privateKey string, passPhrase string) (ssh.AuthMethod, error) {
	buffer := []byte(privateKey)

	if passPhrase == "" {
		key, err := ssh.ParsePrivateKey(buffer)
		if err != nil {
			glog.Errorf("could not parse private key", err)
			return nil, err
		}
		return ssh.PublicKeys(key), nil
	}

	key, err := ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(passPhrase))
	if err != nil {
		glog.Errorf("could not parse private key with passphrase", err)
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

// this should allow local use of clusterctl to access remote cluster as long as your socket
// has the private key added to the agent.
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}
