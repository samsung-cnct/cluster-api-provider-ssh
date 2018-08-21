package machine

import (
	"testing"
)

func TestNewConfigWatch(t *testing.T) {
	var testcases = []struct {
		name         string
		valid        bool
		expectedPath string
	}{
		{
			name:         "valid test file",
			valid:        true,
			expectedPath: "./test_files/fake_machine_setup.yaml",
		},
		{
			name:         "invalid test file",
			valid:        false,
			expectedPath: "this/path/does/not/exist",
		},
	}

	for _, tc := range testcases {
		cw, err := NewConfigWatch(tc.expectedPath)
		if err != nil {
			if tc.name != "invalid test file" {
				t.Errorf("unexpected error: for test case %s, could not create ConfigWatch", tc.name)
			}
		}
		if tc.valid && cw.path != tc.expectedPath {
			t.Errorf("error: for test case %s, ConfigWatch.path should be %s", tc.name, tc.expectedPath)
		}
	}
}

func TestGetMachineSetupConfig(t *testing.T) {
	var testcases = []struct {
		name  string
		valid bool
		cw    ConfigWatch
	}{
		{
			name:  "valid test config path",
			valid: true,
			cw:    ConfigWatch{path: "./test_files/fake_machine_setup.yaml"},
		},
		{
			name:  "valid test config path",
			valid: false,
			cw:    ConfigWatch{path: "/no/such/path"},
		},
	}
	for _, tc := range testcases {
		mc, err := tc.cw.GetMachineSetupConfig()
		if err != nil && tc.valid {
			t.Errorf("unexpected error: for test case %s, could not get MachineSetupConfig", tc.name)
		}
		if !tc.valid && mc != nil {
			t.Errorf("unexpected error: for test case %s, expected MachineSetupConfig to be nil", tc.name)
		}
	}
}
