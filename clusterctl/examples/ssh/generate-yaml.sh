#!/bin/sh
set -e

OUTPUT_DIR=out
mkdir -p ${OUTPUT_DIR}

BOOTSTRAP_DIR=bootstrap_scripts

# --- MACHINES ---
MACHINE_TEMPLATE_FILE=machines.yaml.template
MACHINE_GENERATED_FILE=${OUTPUT_DIR}/machines.yaml


# --- CLUSTERS ---
CLUSTER_TEMPLATE_FILE=cluster.yaml.template
CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/cluster.yaml

# --- PROVIDER_CONFIG ---
PROVIDERCOMPONENT_TEMPLATE_FILE=provider-components.yaml.template
PROVIDERCOMPONENT_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml

PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_BOOTSTRAP=${BOOTSTRAP_DIR}/master_ubuntu_16.04.template
PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_BOOTSTRAP=${BOOTSTRAP_DIR}/node_ubuntu_16.04.template

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

if [ -z ${CLUSTER_PRIVATE_KEY_PLAIN+x} ]; then
    echo "Please enter a valid Cluster Private Key"
    exit 1
fi

if [ -z ${CLUSTER_PASSPHRASE+x} ]; then
    echo "using empty pass phrase to private key"
    CLUSTER_PASSPHRASE=""
fi

# Variables that need to be base64 encoded (for secrets)
OS=$(uname)
if [[ "$OS" =~ "Linux" ]]; then
    CLUSTER_PRIVATE_KEY=$(echo -n $CLUSTER_PRIVATE_KEY_PLAIN | base64 -w0)
elif [[ "$OS" =~ "Darwin" ]]; then
    CLUSTER_PRIVATE_KEY=$(echo -n $CLUSTER_PRIVATE_KEY_PLAIN | base64 | tr -d \\r\\n)
else
  echo "Unrecognized OS : $OS"
  exit 1
fi

cat $PROVIDERCOMPONENT_TEMPLATE_FILE \
  | sed -e "s/\$CLUSTER_PRIVATE_KEY/$CLUSTER_PRIVATE_KEY/" \
  | sed -e "s/\$CLUSTER_PASSPHRASE/$CLUSTER_PASSPHRASE/" \
  | sed -e "/\$MASTER_BOOTSTRAP_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_MASTER_BOOTSTRAP" \
  | sed -e "/\$MASTER_BOOTSTRAP_SCRIPT/d" \
  | sed -e "/\$NODE_BOOTSTRAP_SCRIPT/r $PROVIDERCOMPONENT_TEMPLATE_FILE_NODE_BOOTSTRAP" \
  | sed -e "/\$NODE_BOOTSTRAP_SCRIPT/d" \
  > $PROVIDERCOMPONENT_GENERATED_FILE
echo "Done generating $PROVIDERCOMPONENT_GENERATED_FILE"

cat $MACHINE_TEMPLATE_FILE \
  > $MACHINE_GENERATED_FILE
echo "Done generating $MACHINE_GENERATED_FILE"

cat $CLUSTER_TEMPLATE_FILE \
  > $CLUSTER_GENERATED_FILE
echo "Done generating $CLUSTER_GENERATED_FILE"

echo "You will still need to _edit_ the cluster.yaml and machines.yaml! See the README.md for details."

