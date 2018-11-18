package ssh

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/golang/glog"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"time"
)

const (
	// TODO: This is to quickly work around a customer problem. We should
	// implement a connection pool instead.
	SshTimeoutSeconds    = 600
	SshTimeout           = time.Duration(SshTimeoutSeconds) * time.Second
	GetKubeconfigCommand = "cat /etc/kubernetes/admin.conf"
)

type SSHProviderClientInterface interface {
	ProcessCMD(cmd string) error
	ProcessCMDWithOutput(cmd string) ([]byte, error)
	WritePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error
	DeletePublicKeys(machineSSHConfig v1alpha1.SSHConfig) error
	GetKubeConfig() (string, error)
	GetKubeConfigBytes() ([]byte, error)
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
	bytes, err := s.ProcessCMDWithOutput(GetKubeconfigCommand)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (s *sshProviderClient) GetKubeConfigBytes() ([]byte, error) {
	bytes, err := s.ProcessCMDWithOutput(GetKubeconfigCommand)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (s *sshProviderClient) ProcessCMD(cmd string) error {
	session, connection, err := GetBasicSession(s)
	if err != nil {
		return fmt.Errorf("failed to create a session: %v", err)
	}
	defer session.Close()
	defer connection.Close()

	outputBytes, err := session.CombinedOutput(cmd)
	if err != nil {
		return err
	}

	glog.Infof("Command output = %s ", string(outputBytes[:]))
	return nil
}

func (s *sshProviderClient) ProcessCMDWithOutput(cmd string) ([]byte, error) {
	session, connection, err := GetBasicSession(s)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()
	defer connection.Close()

	outputBytes, err := session.Output(cmd)

	return outputBytes, err
}

func (s *sshProviderClient) WriteFile(scriptLines string, remotePath string) error {
	session, connection, err := GetBasicSession(s)
	if err != nil {
		return fmt.Errorf("failed to create a session: %v", err)
	}

	defer session.Close()
	defer connection.Close()

	// create temporary file
	tempFile, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	// copy script lines into file
	if _, err = tempFile.Write([]byte(scriptLines)); err != nil {
		return err
	}

	// scp over to host
	err = scp.CopyPath(tempFile.Name(), remotePath, session)
	if err != nil {
		return err
	}

	return nil
}

func GetBasicSession(s *sshProviderClient) (*ssh.Session, *ssh.Client, error) {
	var sshConfig *ssh.ClientConfig
	sshAuthMethods := make([]ssh.AuthMethod, 0)

	if s.privateKey != "" {
		publicKeyMethod, err := PublicKeyFile(s.privateKey, s.passPhrase)
		if err != nil {
			return nil, nil, err
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
			// TODO: Host key checking is required to guard against
			// MITM attacks.
			return nil
		},
		Timeout: SshTimeout,
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.address, s.port), sshConfig)
	if err != nil {
		emsg := fmt.Sprintf("failed to dial to %s:%d:", s.address, s.port)
		return nil, nil, fmt.Errorf(emsg, err)
	}

	session, err := connection.NewSession()
	if err != nil {
		glog.Errorf("failed to create sesssion", err)
		return nil, nil,fmt.Errorf("failed to create session: %v", err)
	}

	return session, connection, nil
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
