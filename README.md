# Kubernetes cluster-api-provider-ssh Project

This repository hosts an implementation of a provider using SSH for the [cluster-api project](https://sigs.k8s.io/cluster-api).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

# Development notes

Imports in the code refer to `sigs.k8s.io/cluster-api*` even though this 
repository lives under the `samsung-cnct` GitHub organization. One way
to build this outside of CI is:

```bash
go get github.com/samsung-cnct/cluster-api-provider-ssh
cd $GOPATH/src/github.com/samsung-cnct/cluster-api-provider-ssh
mkdir -p $GOPATH/src/sigs.k8s.io/
ln -s $(pwd) $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
make depend
make
```

## Generating cluster, machine, and provider-components files.

```bash
cd clusterctl/examples/ssh
./generate-yaml.sh
```

For now some manual changes will have to be made to the template files as needed.

## Running cluster deployer
Build the clusterctl binary
```bash
 make compile
```

- Run using minikube:
```bash
./bin/clusterctl create cluster --provider ssh -c ./clusterctl/examples/ssh/out/cluster.yaml -m ./clusterctl/examples/ssh/out/machines.yaml -p ./clusterctl/examples/ssh/out/provider-components.yaml
```

- Run using external cluster:
```bash
./bin/clusterctl create cluster --existing-bootstrap-cluster-kubeconfig /path/to/kubeconfig --provider ssh -c ./clusterctl/examples/ssh/out/cluster.yaml -m ./clusterctl/examples/ssh/out/machines.yaml -p ./clusterctl/examples/ssh/out/provider-components.yaml

```
