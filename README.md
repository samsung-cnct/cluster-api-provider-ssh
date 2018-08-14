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

Follow the instructions [here](./clusterctl/examples/ssh/README.md).

## Deploying a cluster

clusterctl needs access to the private key in order to finalize the new internal cluster.

```bash
eval $(ssh-agent)
ssh-add <private key file>
```

Build the clusterctl binary:

```bash
 make compile
```

- Run using minikube<sup>[1](#kvm2)</sup>:

```bash
bin/clusterctl create cluster --provider ssh \
    -c ./clusterctl/examples/ssh/out/cluster.yaml \
    -m ./clusterctl/examples/ssh/out/machines.yaml \
    -p ./clusterctl/examples/ssh/out/provider-components.yaml
```

- Run using external cluster:

```bash
./bin/clusterctl create cluster --provider ssh \
    --existing-bootstrap-cluster-kubeconfig /path/to/kubeconfig \
    -c ./clusterctl/examples/ssh/out/cluster.yaml \
    -m ./clusterctl/examples/ssh/out/machines.yaml \
    -p ./clusterctl/examples/ssh/out/provider-components.yaml
```

Validate your new cluster:

```bash
export KUBECONFIG=${PWD}/kubeconfig
kubectl get nodes
```

## Building and deploying new controller images for development

To test custom changes to either of the machine controller or the cluster controller, you
need to build and push new images to a repository. There are `make` targets to
do this.

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

* https://quay.io/repository/samsung_cnct/ssh-cluster-controller?tab=tags
* https://quay.io/repository/samsung_cnct/ssh-mchine-controller?tab=tags

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

---

<a name="kvm2">1</a> If using minikube on linux, you may prefer to use the
[kvm2 driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#kvm2-driver).
To do so, add the `--vm-driver=kvm2` flag after installing the driver.

## Cleaning External Cluster for Dev Work
There is a tool to help you dev quickly and redeploy with clean states, the script is located in the hack directory.
First a few assumptions:

- Using pivoting. 
- Using an external cluster such as minikube, or your own.
- minikube/external cluster has kubeconfig file either exported in env variable or in the default directory.

So first create a cluster as part of your dev cycle

```
$ ./bin/clusterctl create cluster -v 7 --provider ssh -c ./clusterctl/examples/ssh/out/cluster.yaml -m ./clusterctl/examples/ssh/out/machines.yaml -p ./clusterctl/examples/ssh/out/provider-components.yaml --existing-bootstrap-cluster-kubeconfig=minikube.kubeconfig
```
Notice that we are using the `--existing-bootstrap-cluster-kubeconfig` flag, that way we do not have to recreate minikube over and over which takes a very long time.
Now, we have finished, pivoted, and completed (whether succeeded or not), next step is to just clean our cluster for the next cycle:

```
$./hack/delete_deployments.sh ssh-controlplane-c97bl new-test-1
Deleting machine object
machine.cluster.k8s.io/ssh-controlplane-c97bl patched
machine.cluster.k8s.io "ssh-controlplane-c97bl" deleted
Deleting cluster object
cluster.cluster.k8s.io/new-test-1 patched
cluster.cluster.k8s.io "new-test-1" deleted
configmap "machine-setup" deleted
configmap "cluster-info" deleted
deployment.extensions "clusterapi-controllers" deleted
deployment.extensions "clusterapi-apiserver" deleted
statefulset.apps "etcd-clusterapi" deleted
```
That should be it, your cluster is good for another run.
