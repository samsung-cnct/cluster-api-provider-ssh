package machine

import "testing"

// TestNewConfigWatch add new tests in test-cases/configwatch.json
func TestNewConfigWatch(t *testing.T) {
  var testcases = []struct {
    name         string
    valid        bool
    path         string
  }{
      {
        name:   "valid test file",
        valid:   true,
        path:   "./test-files/fake_machine_setup.yaml",
      },
      {
        name:   "invalid test file",
        valid:   false,
        path:   "this/path/does/not/exist",
      },
    }

  for _, tc := range testcases {

    cw, err := NewConfigWatch(tc.path)

    if err != nil && tc.valid {
      t.Error("Unexpected error: could not create ConfigWatch for test ")
      t.Errorf("case: '%s' error: '%s'", tc.name, err)
    }

    if tc.valid && cw.path != tc.path {
      t.Errorf("Error: wrong path for ConfigWatch.path for test case '%s'. ", tc.name)
      t.Errorf("Should be '%s', but was '%s'.", tc.path, cw.path)
    }
  }
}

func TestGetMachineSetupConfig(t *testing.T) {
  var testcases = []struct {
    name         string
    valid        bool
    cw           ConfigWatch
  }{
      {
        name:   "valid test config path",
        valid:   true,
        cw:      ConfigWatch{path: "./test-files/fake_machine_setup.yaml"},
      },
      {
        name:   "invalid test config path",
        valid:   false,
        cw:      ConfigWatch{path: "/no/such/path"},
      },
    }

  for _, tc := range testcases {

    machine, err := tc.cw.GetMachineSetupConfig()

    if err != nil && tc.valid {
      t.Error("Unexpected error: could not get MachineSetupConfig:")
      t.Errorf("case: '%s' error: '%s'", tc.name, err)
    }

    if !tc.valid && machine != nil {
      t.Errorf("Unexpected error: expected MachineSetupConfig to be nil for test case %s.", tc.name)
    }
  }
}
