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

- Run the script:

```bash
./generate-yaml.sh
```

If yaml file already exists, you will see an error like the one below:

```bash
$ ./generate-yaml.sh
File provider-components.yaml already exists. Delete it manually before running this script.
```

Update the `out/machines.yaml` file with ip addresses of ubuntu 16.04 nodes
in the `spec.providerConfig.value.sshConfig.host` field. Do this for each
machine that will be part of the new cluster.
