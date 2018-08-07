# Introduction

Three manifest files are necessary to create a Cluster API defined k8s cluster:

- `cluster.yaml`: Defines cluster networking (e.g. pod and service cidrs, etc.).
- `machine.yaml`: Defines k8s versions, ssh configuration, etc.
- `providerconfig.yaml`: Defines Cluster API controller deployments, ssh key
secrets, bootstrap scripts, etc.

## Usage

- Generate an ssh key for this cluster by running `ssh-keygen` and encode it:

```
ssh-keygen -t rsa -b 2048 -f ./id_rsa -N ""
cat id_rsa | base64 | tr -d \\r\\n > id_rsa.b64
```

- Set environment variables

```
export CLUSTER_PRIVATE_KEY=$(cat id_rsa.b64)
```

- Run the script:

```
./generate-yaml.sh
```

If yaml file already exists, you will see an error like the one below:

```
$ ./generate-yaml.sh
File provider-components.yaml already exists. Delete it manually before running this script.
```

You may always manually curate files based on the examples provided. At the 
very least you _must_ update the IP addresses to match the nodes which should
be provisioned with k8s.
