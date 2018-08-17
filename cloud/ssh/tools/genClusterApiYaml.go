package main

import (
	"fmt"

	"github.com/golang/glog"
	"sigs.k8s.io/cluster-api/clusterctl/clusterdeployer"
)

func main() {
	// TODO: remove vendor files and update if moves to pkg dir
	// https://github.com/kubernetes-sigs/cluster-api/pull/477
	yaml, err := clusterdeployer.GetApiServerYaml()
	if err != nil {
		glog.Error(err)
	}
	fmt.Println(yaml)
}
