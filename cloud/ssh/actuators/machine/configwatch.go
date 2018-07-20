package machine

import "os"

// Config Watch holds the path to the named machines yaml file.
type ConfigWatch struct {
	path string
}

func NewConfigWatch(path string) (*ConfigWatch, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &ConfigWatch{path: path}, nil
}

func (cw *ConfigWatch) GetMachineSetupConfig() (MachineSetupConfig, error) {
	file, err := os.Open(cw.path)
	if err != nil {
		return nil, err
	}
	return parseMachineSetupYaml(file)
}
