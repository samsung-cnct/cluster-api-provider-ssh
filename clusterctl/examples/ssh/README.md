# Introduction

Three manifest files are necessary to create a Cluster API defined k8s cluster:

- `cluster.yaml`: Defines cluster networking (e.g. pod and service cidrs, etc.).
- `machine.yaml`: Defines k8s versions, ssh configuration, etc.
- `provider-components.yaml`: Defines Cluster API controller deployments, ssh key
  secrets, bootstrap scripts, etc.

## Usage

From the `clusterctl/examples/ssh` directory

- Generate an ssh key for this cluster by running `ssh-keygen` and encode it:

```bash
ssh-keygen -t ecdsa -b 256 -f ./id_ecdsa -N ""
cat id_ecdsa | base64 | tr -d \\r\\n > id_ecdsa.b64
```

- Set environment variables

```bash
export CLUSTER_PRIVATE_KEY=$(cat id_ecdsa.b64)
```

- Run the generate-yaml.sh script:

For ubuntu:

```bash
export OS_TYPE=ubuntu
./generate-yaml.sh
```

For _air gapped_ centos:

```bash
export OS_TYPE=centos
./generate-yaml.sh
```

Note: The current centos bootstrap scripts are very
environment specific due to an air gap requirement:

- `clusterctl/examples/ssh/bootstrap_scripts/master_bootstrap_air_gapped_centos_7.template`
- `clusterctl/examples/ssh/bootstrap_scripts/node_bootstrap_air_gapped_centos_7.template`

If yaml file already exists, you will see an error like the one below:

<!-- markdownlint-disable MD013 -->

```bash
$ ./generate-yaml.sh
File provider-components.yaml already exists. Delete it manually before running this script.
```

<!-- markdownlint-enable MD013 -->

Update the `out/machines.yaml` file with ip addresses of ubuntu 16.04 nodes
in the `spec.providerConfig.value.sshConfig.host` field.
These machines must be in the same network as your developer machine, which can
be accomplished in multiple ways that would be too extensive to cover here.
Do this for each machine that will be part of the new cluster.
