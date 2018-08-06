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

## Obtaining the code

Imports in the code refer to `sigs.k8s.io/cluster-api*` even though this
repository lives under the `samsung-cnct` GitHub organization. For Go dependencies to be built correctly with `dep`, place this repository in your $GOPATH as follows:

```bash
mkdir -p $GOPATH/src/sigs.k8s.io/
git clone https://github.com/samsung-cnct/cluster-api-provider-ssh.git $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
make
```

## Generating cluster, machine, and provider-components files.

Follow the instructions [here][./clusterctl/examples/ssh/README.md].

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

## Building and deploying new controller images for developement

When making changes to the either of the controllers, to test them you
need to build and push new images to quay.io. There are `Makefile`s to
do this for development (for production CI/CD will handle this).  The Makefile
located at cluster-api-provider-ssh will pass through to the Makefiles in the cmd dir.
For example:

- push both ssh-cluster-controller and ssh-machine-controller images
```
cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
make dev_push

```

- push ssh-machine-controller image
```
cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
make dev_push_machine
```
- push ssh-cluster-controller image
```
cd $GOPATH/src/sigs.k8s.io/cluster-api-provider-ssh
make dev_push_cluster
```

The images will be tagged with the username of the account you used to
build and push the images:

https://quay.io/repository/samsung_cnct/ssh-cluster-controller?tab=tags
https://quay.io/repository/samsung_cnct/ssh-mchine-controller?tab=tags

Remember to change the `provider-components.yaml` manifest to point to your
images. For example:

```
diff --git a/clusterctl/examples/ssh/provider-components.yaml.template b/clusterctl/examples/ssh/provider-components.yaml.template
index 8fac530..3d6c246 100644
--- a/clusterctl/examples/ssh/provider-components.yaml.template
+++ b/clusterctl/examples/ssh/provider-components.yaml.template
@@ -45,7 +45,7 @@ spec:
             cpu: 100m
             memory: 30Mi
       - name: ssh-cluster-controller
-        image: gcr.io/k8s-cluster-api/ssh-cluster-controller:0.0.1
+        image: gcr.io/k8s-cluster-api/ssh-cluster-controller:paul
         volumeMounts:
           - name: config
             mountPath: /etc/kubernetes
@@ -69,7 +69,7 @@ spec:
             cpu: 400m
             memory: 500Mi
       - name: ssh-machine-controller
-        image: gcr.io/k8s-cluster-api/ssh-machine-controller:0.0.1
+        image: gcr.io/k8s-cluster-api/ssh-machine-controller:paul
         volumeMounts:
           - name: config
             mountPath: /etc/kubernetes
```
