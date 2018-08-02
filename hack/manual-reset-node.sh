#!/usr/bin/env bash

# This script is intended to be used on a node that one wishes to remove from a cluster
# Please use wisely and only when needed to clean a machine from its use

node=$1


if [ -z ${node+x} ]; then
  echo 'please enter a valid node'
  exit 1
fi

sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf drain ${node} --delete-local-data --force --ignore-daemonsets
sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf delete ${node}
sudo kubeadm reset

sudo rm -rf /var/lib/etcd

echo "removing all installed packages"
sudo apt-get purge kubeadm kubectl kubelet kubernetes-cni kube* -y