#!/bin/bash

# To be used when using either minikube or an external cluster
# deletes resources created by using clusterctl WITH pivoting
# If not using pivoting it may NOT be a good idea to do this
# Since in that case machine and cluster resources may in fact
# exist in a way that delete would and should actually delete these.

machine=$1
cluster=$2


echo "Deleting machine object"
kubectl patch machine ${machine} -p '{"metadata":{"finalizers": [null]}}'
kubectl delete machine ${machine}
kubectl delete secret master-private-key
kubectl delete secret node-private-key

echo "Deleting cluster object"
kubectl patch cluster ${cluster} -p '{"metadata":{"finalizers": [null]}}'
kubectl delete cluster ${cluster}

kubectl delete configmap machine-setup -n default
kubectl delete configmap cluster-info -n kube-public

kubectl delete deployment clusterapi-controllers
kubectl delete deployment clusterapi-apiserver
kubectl delete statefulsets  etcd-clusterapi
kubectl delete secret cluster-apiserver-certs