package machine

import (
	"text/template"

	"bytes"

	"errors"
	"fmt"

	"github.com/golang/glog"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var (
	masterEnvironmentVarsTemplate *template.Template
	nodeEnvironmentVarsTemplate   *template.Template
)

type Metadata struct {
	StartupScript  string `json:"startupScript"`
	ShutdownScript string `json:"shutdownScript"`
	Items          map[string]string
}

func init() {
	masterEnvironmentVarsTemplate = template.Must(template.New("masterEnvironmentVars").Parse(masterEnvironmentVars))
	nodeEnvironmentVarsTemplate = template.Must(template.New("nodeEnvironmentVars").Parse(nodeEnvironmentVars))
}

type metadataParams struct {
	Token    string
	Cluster  *clusterv1.Cluster
	Machine  *clusterv1.Machine
	Metadata *Metadata

	// These fields are set when executing the template if they are necessary.
	PodCIDR        string
	ServiceCIDR    string
	MasterEndpoint string
}

func masterMetadata(c *clusterv1.Cluster, m *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	params := metadataParams{
		Cluster:     c,
		Machine:     m,
		Metadata:    metadata,
		PodCIDR:     getSubnet(c.Spec.ClusterNetwork.Pods),
		ServiceCIDR: getSubnet(c.Spec.ClusterNetwork.Services),
	}
	masterMetadata := map[string]string{}
	var buf bytes.Buffer

	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	masterMetadata["startup-script"] = buf.String()

	buf.Reset()
	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.ShutdownScript)
	masterMetadata["shutdown-script"] = buf.String()

	return masterMetadata, nil
}

func nodeMetadata(token string, c *clusterv1.Cluster, m *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	nodeMetadata := map[string]string{}
	if len(c.Status.APIEndpoints) < 1 {
		return nodeMetadata, errors.New("The master APIEndpoints has not been initialized in ClusterStatus")
	}
	params := metadataParams{
		Token:          token,
		Cluster:        c,
		Machine:        m,
		Metadata:       metadata,
		PodCIDR:        getSubnet(c.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    getSubnet(c.Spec.ClusterNetwork.Services),
		MasterEndpoint: getEndpoint(c.Status.APIEndpoints[0]),
	}
	glog.Infof("nodeMetadata: params.MasterEndpoint = ", params.MasterEndpoint)

	var buf bytes.Buffer

	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	nodeMetadata["startup-script"] = buf.String()

	buf.Reset()
	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.ShutdownScript)
	nodeMetadata["shutdown-script"] = buf.String()

	return nodeMetadata, nil
}

func getEndpoint(apiEndpoint clusterv1.APIEndpoint) string {
	return fmt.Sprintf("%s:%d", apiEndpoint.Host, apiEndpoint.Port)
}

// Just a temporary hack to grab a single range from the config.
func getSubnet(netRange clusterv1.NetworkRanges) string {
	if len(netRange.CIDRBlocks) == 0 {
		return ""
	}
	return netRange.CIDRBlocks[0]
}

const masterEnvironmentVars = `#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
VERSION=v${KUBELET_VERSION}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE_NAME={{ .Machine.ObjectMeta.Name }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+=$MACHINE_NAME
CONTROL_PLANE_VERSION={{ .Machine.Spec.Versions.ControlPlane }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
`
const nodeEnvironmentVars = `#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
TOKEN={{ .Token }}
MASTER={{ .MasterEndpoint }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE_NAME={{ .Machine.ObjectMeta.Name }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+=$MACHINE_NAME
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
`
