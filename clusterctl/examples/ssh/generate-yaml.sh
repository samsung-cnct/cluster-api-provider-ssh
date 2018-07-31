#!/bin/sh
set -e

OUTPUT_DIR=out
mkdir -p ${OUTPUT_DIR}

MACHINE_TEMPLATE_FILE=machines.yaml.template
MACHINE_GENERATED_FILE=${OUTPUT_DIR}/machines.yaml
CLUSTER_TEMPLATE_FILE=cluster.yaml.template
CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/cluster.yaml
PROVIDERCOMPONENT_TEMPLATE_FILE=provider-components.yaml.template
PROVIDERCOMPONENT_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml

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
    echo "Please generate a valid Cluster Private Key. Then run:"
    echo "export CLUSTER_PRIVATE_KEY_PLAIN=$(cat `path/to/private/key/file`)"
    exit 1
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
  > $PROVIDERCOMPONENT_GENERATED_FILE
echo "Done generating $PROVIDERCOMPONENT_GENERATED_FILE"

cat $MACHINE_TEMPLATE_FILE \
  > $MACHINE_GENERATED_FILE
echo "Done generating $MACHINE_GENERATED_FILE"

cat $CLUSTER_TEMPLATE_FILE \
  > $CLUSTER_GENERATED_FILE
echo "Done generating $CLUSTER_GENERATED_FILE"

echo "You will still need to _edit_ the cluster.yaml and machines.yaml! See the README.md for details."

