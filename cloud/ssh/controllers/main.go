package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/util/logs"

	"sigs.k8s.io/cluster-api/pkg/controller/config"
	machineOptions "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/machine/options"
	machineController "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/machine"
	clusterOptions "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/cluster/options"
	clusterController "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/cluster"

)

func main() {

	fs := pflag.CommandLine
	var controllerType, machineSetupConfigsPath string
	fs.StringVar(&controllerType, "controller", controllerType, "specify whether this should run the machine or cluster controller")
	fs.StringVar(&machineSetupConfigsPath, "machinesetup", machineSetupConfigsPath, "path to machine setup configs file")
	config.ControllerConfig.AddFlags(pflag.CommandLine)
	// the following line exists to make glog happy, for more information, see: https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})
	pflag.Parse()

	logs.InitLogs()
	defer logs.FlushLogs()

	switch (controllerType) {
	case "machine":
		machineServer := machineOptions.NewServer(machineSetupConfigsPath)
		if err := machineController.Run(machineServer); err != nil {
			glog.Errorf("Failed to start machine controller. Err: %v", err)
		}

	case "cluster":
		clusterServer := clusterOptions.NewServer(machineSetupConfigsPath)
		if err := clusterController.Run(clusterServer); err != nil {
			glog.Errorf("Failed to start cluster controller. Err: %v", err)
		}
	default:
		glog.Errorf("Failed to start controller, `controller` flag must be either `machine` or `cluster` but was %v.", controllerType)
	}
}


