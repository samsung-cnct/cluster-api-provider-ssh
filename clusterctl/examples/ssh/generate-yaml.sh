#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

OUTPUT_DIR=out
mkdir -p ${OUTPUT_DIR}

BOOTSTRAP_DIR=bootstrap_scripts

OS_TYPE=${1:-ubuntu}

# --- MACHINES ---
MACHINE_TEMPLATE_FILE=machines.yaml.template
MACHINE_GENERATED_FILE=${OUTPUT_DIR}/machines.yaml


# --- CLUSTERS ---
CLUSTER_TEMPLATE_FILE=cluster.yaml.template
CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/cluster.yaml

# --- PROVIDER_CONFIG ---
PROVIDERCOMPONENT_TEMPLATE_FILE=provider-components.yaml.template
PROVIDERCOMPONENT_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml

if [ $OS_TYPE == "ubuntu" ]; then
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_BOOTSTRAP=${BOOTSTRAP_DIR}/master_bootstrap_ubuntu_16.04.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_BOOTSTRAP=${BOOTSTRAP_DIR}/node_bootstrap_ubuntu_16.04.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_TEARDOWN=${BOOTSTRAP_DIR}/master_teardown_ubuntu_16.04.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_TEARDOWN=${BOOTSTRAP_DIR}/node_teardown_ubuntu_16.04.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_UPGRADE=${BOOTSTRAP_DIR}/master_upgrade_ubuntu_16.04.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_UPGRADE=${BOOTSTRAP_DIR}/node_upgrade_ubuntu_16.04.template
else 
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_BOOTSTRAP=${BOOTSTRAP_DIR}/master_bootstrap_centos_7.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_BOOTSTRAP=${BOOTSTRAP_DIR}/node_bootstrap_centos_7.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_TEARDOWN=${BOOTSTRAP_DIR}/master_teardown_centos_7.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_TEARDOWN=${BOOTSTRAP_DIR}/node_teardown_centos_7.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_UPGRADE=${BOOTSTRAP_DIR}/master_upgrade_centos_7.template
    PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_UPGRADE=${BOOTSTRAP_DIR}/node_upgrade_centos_7.template
fi

OVERWRITE=0

SCRIPT=$(basename $0)
while test $# -gt 0; do
        case "$1" in
          -h|--help)
            echo "$SCRIPT - generates input yaml files for Cluster API on openstack"
            echo " "
            echo "$SCRIPT [options]"
            echo " "
            echo "options:"
            echo "-h, --help                show brief help"
            echo "-f, --force-overwrite     if file to be generated already exists, force script to overwrite it"
            exit 0
            ;;
          -f)
            OVERWRITE=1
            shift
            ;;
          --force-overwrite)
            OVERWRITE=1
            shift
            ;;
          *)
            break
            ;;
        esac
done

if [ $OVERWRITE -ne 1 ] && [ -f $PROVIDERCOMPONENT_GENERATED_FILE ]; then
  echo "File $PROVIDERCOMPONENT_GENERATED_FILE already exists. Delete it manually before running this script."
  exit 1
fi

# TODO Fill out the generation pieces as we need them.

if [ -z ${CLUSTER_PRIVATE_KEY+x} ]; then
    echo "Please generate a valid base64 encoded cluster private key and export the key file contents to CLUSTER_PRIVATE_KEY."
    exit 1
fi

if [ -z ${CLUSTER_PASSPHRASE+x} ]; then
    echo "using empty cluster pass phrase to private key"
    CLUSTER_PASSPHRASE='""'
fi

cat $PROVIDERCOMPONENT_TEMPLATE_FILE \
  | sed -e "s/\$CLUSTER_PRIVATE_KEY/$CLUSTER_PRIVATE_KEY/" \
  | sed -e "s/\$CLUSTER_PASSPHRASE/$CLUSTER_PASSPHRASE/" \
  | sed -e "/\$MASTER_BOOTSTRAP_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_BOOTSTRAP" \
  | sed -e "/\$MASTER_BOOTSTRAP_SCRIPT/d" \
  | sed -e "/\$NODE_BOOTSTRAP_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_BOOTSTRAP" \
  | sed -e "/\$NODE_BOOTSTRAP_SCRIPT/d" \
  | sed -e "/\$MASTER_TEARDOWN_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_TEARDOWN" \
  | sed -e "/\$MASTER_TEARDOWN_SCRIPT/d" \
  | sed -e "/\$NODE_TEARDOWN_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_TEARDOWN" \
  | sed -e "/\$NODE_TEARDOWN_SCRIPT/d" \
  | sed -e "/\$MASTER_UPGRADE_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_UPGRADE" \
  | sed -e "/\$MASTER_UPGRADE_SCRIPT/d" \
  | sed -e "/\$NODE_UPGRADE_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_UPGRADE" \
  | sed -e "/\$NODE_UPGRADE_SCRIPT/d" \
  > $PROVIDERCOMPONENT_GENERATED_FILE
echo "Done generating $PROVIDERCOMPONENT_GENERATED_FILE"

cat $MACHINE_TEMPLATE_FILE \
  > $MACHINE_GENERATED_FILE
echo "Done generating $MACHINE_GENERATED_FILE"

cat $CLUSTER_TEMPLATE_FILE \
  > $CLUSTER_GENERATED_FILE
echo "Done generating $CLUSTER_GENERATED_FILE"

echo "You will still need to _edit_ the cluster.yaml and machines.yaml! See the README.md for details."

