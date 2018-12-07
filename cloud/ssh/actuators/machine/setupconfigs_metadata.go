package machine

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/samsung-cnct/cluster-api-provider-ssh/cloud/ssh/providerconfig/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var (
	masterEnvironmentVarsTemplate *template.Template
	nodeEnvironmentVarsTemplate   *template.Template
)

const (
	startupScriptKey  = "startup-script"
	shutdownScriptKey = "shutdown-script"
	upgradeScriptKey  = "upgrade-script"
)

type Metadata struct {
	StartupScript  string `json:"startupScript"`
	ShutdownScript string `json:"shutdownScript"`
	UpgradeScript  string `json:"upgradeScript"`
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
	MasterEndpoint string // for node joining a cluster, should be available after master created
	MasterIP       string // for injection to startup script
	NodeLabels     string
}

func masterMetadata(c *clusterv1.Cluster, m *clusterv1.Machine, metadata *Metadata, sshConfig v1alpha1.SSHConfig) (map[string]string, error) {
	// The Cluster API Machine controller adds machine labels to nodes. Since
	// our Machines do not exist in the managed cluster we have to manually
	// add the labels from the bootstrapping scripts.
	nodeLabelList := make([]string, 0)
	for k, v := range m.Labels {
		nodeLabelList = append(nodeLabelList, fmt.Sprintf("%s=%s", k, v))
	}
	nodeLabels := strings.Join(nodeLabelList, " ")

	params := metadataParams{
		Cluster:     c,
		Machine:     m,
		Metadata:    metadata,
		PodCIDR:     getSubnet(c.Spec.ClusterNetwork.Pods),
		ServiceCIDR: getSubnet(c.Spec.ClusterNetwork.Services),
		MasterIP:    sshConfig.Host,
		NodeLabels:  nodeLabels,
	}
	masterMetadata := map[string]string{}
	var buf bytes.Buffer

	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	masterMetadata[startupScriptKey] = buf.String()

	buf.Reset()
	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.ShutdownScript)
	masterMetadata[shutdownScriptKey] = buf.String()

	buf.Reset()
	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.UpgradeScript)
	masterMetadata[upgradeScriptKey] = buf.String()

	return masterMetadata, nil
}

func nodeMetadata(token string, c *clusterv1.Cluster, m *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	nodeMetadata := map[string]string{}

	if len(c.Status.APIEndpoints) < 1 {
		return nodeMetadata, errors.New("The master APIEndpoints has not been initialized in ClusterStatus")
	}

	// The Cluster API Machine controller adds machine labels to nodes. Since
	// our Machines do not exist in the managed cluster we have to manually
	// add the labels from the bootstrapping scripts.
	nodeLabelList := make([]string, 0)
	for k, v := range m.Labels {
		nodeLabelList = append(nodeLabelList, fmt.Sprintf("%s=%s", k, v))
	}
	nodeLabels := strings.Join(nodeLabelList, " ")

	params := metadataParams{
		Token:          token,
		Cluster:        c,
		Machine:        m,
		Metadata:       metadata,
		PodCIDR:        getSubnet(c.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    getSubnet(c.Spec.ClusterNetwork.Services),
		MasterEndpoint: getEndpoint(c.Status.APIEndpoints[0]),
		NodeLabels:     nodeLabels,
	}
	glog.Infof("The MasterEndpoint = %s, machine %s cluster %s", params.MasterEndpoint, m.Name, c.Name)
	var buf bytes.Buffer

	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	nodeMetadata[startupScriptKey] = buf.String()

	buf.Reset()
	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.ShutdownScript)
	nodeMetadata[shutdownScriptKey] = buf.String()

	buf.Reset()
	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.UpgradeScript)
	nodeMetadata[upgradeScriptKey] = buf.String()

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

const masterEnvironmentVars = `#!/usr/bin/env bash
CONTROL_PLANE_VERSION={{ .Machine.Spec.Versions.ControlPlane }}
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
MASTER_IP={{ .MasterIP }}
NODE_LABELS="{{ .NodeLabels }}"
`
const nodeEnvironmentVars = `#!/usr/bin/env bash
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
NODE_LABELS="{{ .NodeLabels }}"
`
