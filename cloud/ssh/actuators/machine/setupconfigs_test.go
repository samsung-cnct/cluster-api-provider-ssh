package machine

import (
	"reflect"
	"testing"

	"sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestGetYaml(t *testing.T) {
	var testcases = []struct {
		name      string
		valid     bool
		confItems ValidMachineConfigItems
	}{
		{
			name:  "Marshaling Yaml",
			valid: true,
			confItems: ValidMachineConfigItems{
				machineConfigList: &MachineConfigList{
					Items: []MachineItem{
						{
							Params: MachineParams{
								Roles: []v1alpha1.MachineRole{
									"Node",
								},
								Versions: clusterv1.MachineVersionInfo{
									Kubelet:      "1.10.6",
									ControlPlane: "1.10.6",
								},
							},
							Metadata: Metadata{
								StartupScript:  "hello",
								ShutdownScript: "goodbye",
								Items:          map[string]string{"unicorns": "awesome", "owls": "great"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		vc := tc.confItems
		result, err := vc.GetYaml()
		if err != nil {
			t.Errorf("unexpected error: for test case %s, failed to get yaml", tc.name)
		}
		s := reflect.TypeOf(result).String()
		if s != "string" {
			t.Errorf("expected return value of GetYaml to be a string for test case %s", tc.name)
		}
	}
}
