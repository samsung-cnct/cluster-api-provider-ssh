package machine

import (
  "encoding/json"
  "io/ioutil"
  "os"
  "path/filepath"
  "testing"
  "fmt"
)

var config CWTestCases

type allTests struct {
  tests  []string
  test   string
}

type CWTestCases struct {
  Testcases   []CWTestCase `json:"testcases"`
}

type CWTestCase struct {
  Name         string `json:"name"`
  Description  string `json:"description"`
  Type         string `json:"type"`
  Valid        bool   `json:"valid"`
  Path         string `json:"path"`
  Cw           ConfigWatch
}

func TestMain(m *testing.M) {

  testDir         := "test-cases"
  testJSONFile, _ := ioutil.ReadFile(filepath.Join(testDir,"configwatch.json"))
  err             := json.Unmarshal(testJSONFile, &config)

  if err != nil {
    err.Error()
  }

  os.Exit(m.Run())
}

// TestNewConfigWatch add new tests in test-cases/configwatch.json
func TestNewConfigWatch(t *testing.T) {
  for _, tc := range config.Testcases {

    if tc.Type == "" {
      t.Errorf("Unexpected error: test case is missing `Type` property.")
    }

    if tc.Type == "configwatch" {
      fmt.Printf("Test case: %v\n", tc.Name)

      cw, err := NewConfigWatch(tc.Path)

      if err != nil && tc.Valid == true {
        t.Errorf("Unexpected error: could not create ConfigWatch for test ")
        t.Errorf("case: '%s' error: '%s'", tc.Name, err)
      }

      if tc.Valid && cw.path != tc.Path {
        t.Errorf("Error: wrong path for ConfigWatch.path for test case '%s'. ", tc.Name)
        t.Errorf("Should be '%s', but was '%s'.", tc.Path, cw.path)
      }
    }
  }
}

func TestGetMachineSetupConfig(t *testing.T) {
  for _, tc := range config.Testcases {

    tc.Cw         = ConfigWatch{path: tc.Path}
    machine, err := tc.Cw.GetMachineSetupConfig()

    if tc.Type == "configmachine" {
      fmt.Printf("Test case: %v\n", tc.Name)

      if err != nil && tc.Valid == true {
        t.Errorf("Unexpected error: could not get MachineSetupConfig:")
        t.Errorf("case: '%s' error: '%s'", tc.Name, err)
      }

      if !tc.Valid && machine != nil {
        t.Errorf("Unexpected error: expected MachineSetupConfig to be nil for test case %s.", tc.Name)
      }
    }
  }
}
