package machine

import (
  "testing"
  "fmt"
)

// TestNewConfigWatch add new tests in test-cases/configwatch.json
func TestNewConfigWatch(t *testing.T) {
  var testcases = []struct {
    name         string
    description  string
    testtype     string
    valid        bool
    path         string
    cw           ConfigWatch
  }{
      {
        name:         "valid test file",
        testtype:     "configwatch",
        description:  "Valid test file - test if valid file exists.",
        valid:         true,
        path:         "./test-files/fake_machine_setup.yaml",
      },
      {
        name:         "invalid test file",
        testtype:     "configwatch",
        description:  "Invalid test file - test if invalid file exists.",
        valid:         false,
        path:         "this/path/does/not/exist",
      },
    }

  for _, tc := range testcases {

    cw, err := NewConfigWatch(tc.path)

    if err != nil && tc.valid {
      t.Errorf("Unexpected error: could not create ConfigWatch for test ")
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
    description  string
    testtype     string
    valid        bool
    path         string
    cw           ConfigWatch
  }{
      {
        name:         "valid test config path",
        testtype:     "configmachine",
        valid:         true,
        path:         "./test_files/fake_machine_setup.yaml",
      },
      {
        name:         "invalid test config path",
        testtype:     "configmachine",
        valid:         false,
        path:         "/no/such/path",
      },
    }

  for _, tc := range testcases {

    tc.cw         = ConfigWatch{path: tc.path}
    machine, err := tc.cw.GetMachineSetupConfig()

    if tc.testtype == "configmachine" {
      if err != nil && tc.valid {
        t.Errorf("Unexpected error: could not get MachineSetupConfig:")
        t.Errorf("case: '%s' error: '%s'", tc.name, err)
      }

      if !tc.valid && machine != nil {
        t.Errorf("Unexpected error: expected MachineSetupConfig to be nil for test case %s.", tc.name)
      }
    }
  }
}
