package machine

import (
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"io"
	"io/ioutil"
)

// MachineSetupConfig defines the methods necessaty for a setup config
type MachineSetupConfig interface {
	GetYaml() (string, error)
	GetMetadata(params *MachineParams) (Metadata, error)
}

// MachineItem holds MachineParams and Metadata
type MachineItem struct {
	//TODO originally this was a list, investigate if we would.
	Params   MachineParams `json:"machineParams"`
	Metadata Metadata      `json:"metadata"`
}

// MachineParams holds api roles and versions for a machine item.
type MachineParams struct {
	Roles    []v1alpha1.MachineRole       `json:"roles"`
	Versions clusterv1.MachineVersionInfo `json:"versions"`
}

// ValidMachineConfigItems refers to the valid machine setup configs parsed
// out of the machine setup configs yaml file held in ConfigWatch.
type ValidMachineConfigItems struct {
	machineConfigList *MachineConfigList
}

// MachineConfigList is a list of MachineItems
type MachineConfigList struct {
	Items []MachineItem `json:"items"`
}

//GetYaml returns a string yaml representation of ValidMachineConfigItems.
func (vc *ValidMachineConfigItems) GetYaml() (string, error) {
	bytes, err := yaml.Marshal(vc.machineConfigList)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

//GetMetadata returns the metadata contained in ValidMachineConfigItems.
func (vc *ValidMachineConfigItems) GetMetadata(params *MachineParams) (Metadata, error) {
	machineSetupConfig, err := vc.matchMachineSetupConfig(params)
	if err != nil {
		return Metadata{}, err
	}
	return machineSetupConfig.Metadata, nil
}

func (vc *ValidMachineConfigItems) matchMachineSetupConfig(params *MachineParams) (*MachineItem, error) {
	matchingConfigs := make([]MachineItem, 0)
	for _, conf := range vc.machineConfigList.Items {
		validParams := conf.Params
		validRoles := rolesToMap(validParams.Roles)
		paramRoles := rolesToMap(params.Roles)
		if !reflect.DeepEqual(paramRoles, validRoles) {
			continue
		}
		if params.Versions != validParams.Versions {
			continue
		}
		matchingConfigs = append(matchingConfigs, conf)
	}

	if len(matchingConfigs) == 1 {
		return &matchingConfigs[0], nil
	} else if len(matchingConfigs) == 0 {
		return nil, fmt.Errorf("could not find a matching machine setup config for params %+v", params)
	} else {
		return nil, fmt.Errorf("found multiple matching machine setup configs for params %+v", params)
	}
}

func parseMachineSetupYaml(reader io.Reader) (*ValidMachineConfigItems, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	configList := &MachineConfigList{}
	err = yaml.Unmarshal(bytes, configList)
	if err != nil {
		return nil, fmt.Errorf("error parsing yaml: %s, %v", string(bytes), err)
	}

	return &ValidMachineConfigItems{machineConfigList: configList}, nil
}

func rolesToMap(roles []v1alpha1.MachineRole) map[v1alpha1.MachineRole]int {
	rolesMap := map[v1alpha1.MachineRole]int{}
	for _, role := range roles {
		rolesMap[role] = rolesMap[role] + 1
	}
	return rolesMap
}
