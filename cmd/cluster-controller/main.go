/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/util/logs"
	"sigs.k8s.io/cluster-api/pkg/controller/config"

	clusterController "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/cluster"
	clusterOptions "sigs.k8s.io/cluster-api-provider-ssh/cloud/ssh/controllers/cluster/options"
)

func main() {
	fs := pflag.CommandLine
	var machineSetupConfigsPath string

	fs.StringVar(&machineSetupConfigsPath, "machinesetup", machineSetupConfigsPath, "path to machine setup configs file")

	config.ControllerConfig.AddFlags(pflag.CommandLine)
	// the following line exists to make glog happy, for more information, see: https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})
	pflag.Parse()

	logs.InitLogs()
	defer logs.FlushLogs()

	clusterServer := clusterOptions.NewServer(machineSetupConfigsPath)
	if err := clusterController.Run(clusterServer); err != nil {
		glog.Errorf("Failed to start cluster controller. Err: %v", err)
	}
}
