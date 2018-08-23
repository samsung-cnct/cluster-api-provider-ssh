package machine

import (
	"os"
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
								StartupScript:  "Guten Tag",
								ShutdownScript: "Auf Wiedersehen",
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
		if tc.valid && s != "string" {
			t.Errorf("expected return value of GetYaml to be a string for test case %s", tc.name)
		}
	}
}

func TestGetMetadata(t *testing.T) {
	var testcases = []struct {
		name           string
		valid          bool
		expectedParams MachineParams
		confItems      ValidMachineConfigItems
	}{
		{
			name:  "Hello Goodbye scripts",
			valid: true,
			expectedParams: MachineParams{
				Roles: []v1alpha1.MachineRole{
					"Node",
				},
				Versions: clusterv1.MachineVersionInfo{
					Kubelet:      "1.10.6",
					ControlPlane: "1.10.6",
				},
			},
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
		{

			name:  "Two same configs",
			valid: false,
			expectedParams: MachineParams{
				Roles: []v1alpha1.MachineRole{
					"Node",
				},
				Versions: clusterv1.MachineVersionInfo{
					Kubelet:      "1.10.6",
					ControlPlane: "1.10.6",
				},
			},
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
		{
			name:  "Invalid Params",
			valid: false,
			expectedParams: MachineParams{
				Roles: []v1alpha1.MachineRole{
					"Node",
				},
				Versions: clusterv1.MachineVersionInfo{
					Kubelet:      "1.10.6",
					ControlPlane: "1.10.6",
				},
			},
			confItems: ValidMachineConfigItems{
				machineConfigList: &MachineConfigList{
					Items: []MachineItem{
						{
							Params: MachineParams{
								Roles: []v1alpha1.MachineRole{
									"Unicorns",
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
		p := tc.expectedParams
		md, err := vc.GetMetadata(&p)
		if err != nil {
			if tc.name == "Two same configs" && err.Error() != "found multiple matching machine setup configs for params &{Roles:[Node] Versions:{Kubelet:1.10.6 ControlPlane:1.10.6}}" {
				t.Errorf("expected multiple matching machine setup configs error for test case %s", tc.name)
			}
			if tc.name == "Invalid Params" && err.Error() != "could not find a matching machine setup config for params &{Roles:[Node] Versions:{Kubelet:1.10.6 ControlPlane:1.10.6}}" {
				t.Errorf("expected could not find matching machine setup config error for test case %s", tc.name)
			}
			if tc.valid {
				t.Errorf("unexpected error: for test case %s, failed to get metadata", tc.name)
			}
		}
		if tc.name == "Hello Goodbye scripts" && md.StartupScript != "hello" {
			t.Errorf("expected metadata startup script for test case %s to equal hello", tc.name)
		}
	}
}

func TestParseMachineSetupYaml(t *testing.T) {
	filename := "./test_files/fake_machine_setup.yaml"
	testfile, err := os.Open(filename)
	if err != nil {
		t.Errorf("unexpected error opening testfile %s", filename)
	}
	vc, err := parseMachineSetupYaml(testfile)
	if err != nil {
		t.Errorf("unexpected error parsing testfile %s", filename)
	}
	testItem := vc.machineConfigList.Items[0].Metadata.Items["unicorns"]
	if testItem != "awesome" {
		t.Error("expected metadata item key \"unicorns\" to have value \"awesome\"")
	}
}
