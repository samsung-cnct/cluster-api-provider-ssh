#!/bin/bash

clusterctl create cluster --vm-driver=none  --provider ssh  -c /config/cluster.yaml -m /config/machines.yaml -p /config/provider-components.yaml 

