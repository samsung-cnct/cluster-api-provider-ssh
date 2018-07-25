package machine

import (
	"text/template"

	"bytes"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"fmt"
)

var (
	masterEnvironmentVarsTemplate *template.Template
	nodeEnvironmentVarsTemplate   *template.Template
	etcdEnvironmentVarsTemplate   *template.Template
)

type Metadata struct {
	StartupScript string `json:"startupScript"`
	Items         map[string]string
}

func init() {
	masterEnvironmentVarsTemplate = template.Must(template.New("masterEnvironmentVars").Parse(masterEnvironmentVars))
	nodeEnvironmentVarsTemplate = template.Must(template.New("nodeEnvironmentVars").Parse(nodeEnvironmentVars))
	etcdEnvironmentVarsTemplate = template.Must(template.New("etcdEnvironmentVars").Parse(etcdEnvironmentVars))
}

type metadataParams struct {
	Token        string
	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	DockerImages []string
	Project      string
	Metadata     *Metadata

	// These fields are set when executing the template if they are necessary.
	PodCIDR        string
	ServiceCIDR    string
	MasterEndpoint string
}

func masterMetadata(cluster *clusterv1.Cluster,
	machine *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	params := metadataParams{
		Cluster:     cluster,
		Machine:     machine,
		Metadata:    metadata,
		PodCIDR:     getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR: getSubnet(cluster.Spec.ClusterNetwork.Services),
	}

	masterMetadata := map[string]string{}
	var buf bytes.Buffer
	if err := masterEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	masterMetadata["startup-script"] = buf.String()
	return masterMetadata, nil
}

func nodeMetadata(token string,
	cluster *clusterv1.Cluster,
	machine *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	params := metadataParams{
		Token:          token,
		Cluster:        cluster,
		Machine:        machine,
		Metadata:       metadata,
		PodCIDR:        getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    getSubnet(cluster.Spec.ClusterNetwork.Services),
		MasterEndpoint: getEndpoint(cluster.Status.APIEndpoints[0]),
	}

	nodeMetadata := map[string]string{}
	var buf bytes.Buffer
	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	nodeMetadata["startup-script"] = buf.String()
	return nodeMetadata, nil
}

func etcdMetadata(cluster *clusterv1.Cluster,
	machine *clusterv1.Machine, metadata *Metadata) (map[string]string, error) {
	params := metadataParams{
		Cluster:        cluster,
		Machine:        machine,
		Metadata:       metadata,
		PodCIDR:        getSubnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    getSubnet(cluster.Spec.ClusterNetwork.Services),
		MasterEndpoint: getEndpoint(cluster.Status.APIEndpoints[0]),
	}

	nodeMetadata := map[string]string{}
	var buf bytes.Buffer
	if err := nodeEnvironmentVarsTemplate.Execute(&buf, params); err != nil {
		return nil, err
	}
	buf.WriteString(params.Metadata.StartupScript)
	nodeMetadata["startup-script"] = buf.String()
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

// TODO(kcoronado): replace with actual network and node tag args when they are added into provider config.
const masterEnvironmentVars = `
#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
VERSION=v${KUBELET_VERSION}
PORT=443
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CONTROL_PLANE_VERSION={{ .Machine.Spec.Versions.ControlPlane }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
# Environment variables for GCE cloud config
NETWORK=default
SUBNETWORK=kubernetes
CLUSTER_NAME={{ .Cluster.Name }}
NODE_TAG="$CLUSTER_NAME-worker"
`

const nodeEnvironmentVars = `
#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
TOKEN={{ .Token }}
MASTER={{ .MasterEndpoint }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
# Environment variables for GCE cloud config
NETWORK=default
SUBNETWORK=kubernetes
CLUSTER_NAME={{ .Cluster.Name }}
NODE_TAG="$CLUSTER_NAME-worker"
`

const etcdEnvironmentVars = `
#!/bin/bash
MASTER={{ .MasterEndpoint }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
# Environment variables for GCE cloud config
NETWORK=default
SUBNETWORK=kubernetes
CLUSTER_NAME={{ .Cluster.Name }}
NODE_TAG="$CLUSTER_NAME-worker"
`
