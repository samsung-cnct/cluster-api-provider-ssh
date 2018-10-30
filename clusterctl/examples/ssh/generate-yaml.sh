#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

inject_functions()
{
  local FUNCTIONS tmpl

  FUNCTIONS=$(< "$bootstrap_dir/common_functions.template")
  tmpl=$1

  export FUNCTIONS

  [[ -z "${tmpl}" ]] && \
    {
      echo >&2 "Usage: preprocess_template(): caller must provide template to process."
      return 45
    }

  # shellcheck disable=SC2016
  envsubst '${FUNCTIONS}' < "$tmpl"
}

runpath()
{
  pushd . > /dev/null
  SCRIPT_PATH="${BASH_SOURCE[0]}"
  if ([ -h "${SCRIPT_PATH}" ]); then
    while([ -h "${SCRIPT_PATH}" ]); do
      cd "$(dirname "$SCRIPT_PATH")";
      SCRIPT_PATH=$(readlink "${SCRIPT_PATH}");
    done
  fi
  cd "$(dirname "${SCRIPT_PATH}")" > /dev/null
  pwd
  popd  > /dev/null
}

generate_yaml()
{
  local bootstrap_dir machine_template_file machine_generated_file cluster_template_file
  local cluster_generated_file providercomponent_template_file providercomponent_generated_file
  local master_upgrade_script master_teardown_script node_upgrade_script node_teardown_script

  if ! mkdir -p "${OUTPUT_DIR}" 2>/dev/null; then
    echo >&2 "Unable to mkdir $OUTPUT_DIR"
    return 12
  fi

  bootstrap_dir=bootstrap_scripts

  # --- MACHINES ---
  machine_template_file="$BASEDIR/machines.yaml.template"
  machine_generated_file=${OUTPUT_DIR}/machines.yaml

  # --- CLUSTERS ---
  cluster_template_file="$BASEDIR/cluster.yaml.template"
  cluster_generated_file=${OUTPUT_DIR}/cluster.yaml

  # --- PROVIDER_CONFIG ---
  providercomponent_template_file="$BASEDIR/provider-components.yaml.template"
  providercomponent_generated_file=${OUTPUT_DIR}/provider-components.yaml

  if [[ $OVERWRITE -ne 1 ]] && [[ -f $providercomponent_generated_file ]]; then
    echo >&1 "File $providercomponent_generated_file already exists. Delete it manually before running this script."
    return 25
  fi

  # This sorely needs optimization. The file naming convention and usage here is not scalable.
  if [[ "${OS_TYPE}" == "ubuntu" ]]; then
    MASTER_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/master_bootstrap_ubuntu_16.04.template)"
    MASTER_TEARDOWN_SCRIPT="$(< ${bootstrap_dir}/master_teardown_ubuntu_16.04.template)"
    MASTER_UPGRADE_SCRIPT="$(< ${bootstrap_dir}/master_upgrade_ubuntu_16.04.template)"

    NODE_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/node_bootstrap_ubuntu_16.04.template)"
    NODE_TEARDOWN_SCRIPT="$(< ${bootstrap_dir}/node_teardown_ubuntu_16.04.template)"
    NODE_UPGRADE_SCRIPT="$(< ${bootstrap_dir}/node_upgrade_ubuntu_16.04.template)"
  else
    if [[ ${SDS_CASE} == 0 ]]; then
      MASTER_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/master_bootstrap_aws_centos_7.template)"

      NODE_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/node_bootstrap_aws_centos_7.template)"
    else
      MASTER_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/master_bootstrap_air_gapped_centos_7.template)"

      NODE_BOOTSTRAP_SCRIPT="$(< ${bootstrap_dir}/node_bootstrap_air_gapped_centos_7.template)"
    fi

    MASTER_TEARDOWN_SCRIPT="$(< ${bootstrap_dir}/master_teardown_centos_7.template)"
    MASTER_UPGRADE_SCRIPT="$(< ${bootstrap_dir}/master_upgrade_centos_7.template)"

    NODE_TEARDOWN_SCRIPT="$(< ${bootstrap_dir}/node_teardown_centos_7.template)"
    NODE_UPGRADE_SCRIPT="$(< ${bootstrap_dir}/node_upgrade_centos_7.template)"
  fi

  # prepend common functions to template script
  FUNCTIONS=$(< "$bootstrap_dir/common_functions.template")

  export CLUSTER_PRIVATE_KEY CLUSTER_PASSPHRASE MASTER_BOOTSTRAP_SCRIPT \
         NODE_BOOTSTRAP_SCRIPT MASTER_TEARDOWN_SCRIPT NODE_TEARDOWN_SCRIPT MASTER_UPGRADE_SCRIPT \
         NODE_UPGRADE_SCRIPT FUNCTIONS OS_TYPE KUBELET_VERSION

  # shellcheck disable=SC2016
  envsubst '$CLUSTER_PRIVATE_KEY $CLUSTER_PASSPHRASE $MASTER_BOOTSTRAP_SCRIPT
            $NODE_BOOTSTRAP_SCRIPT  $MASTER_TEARDOWN_SCRIPT $NODE_TEARDOWN_SCRIPT
            $MASTER_UPGRADE_SCRIPT $NODE_UPGRADE_SCRIPT $KUBELET_VERSION $OS_TYPE' \
           < "$providercomponent_template_file" > "$providercomponent_generated_file-tmp"

  if inject_functions "$providercomponent_generated_file-tmp" > "$providercomponent_generated_file"; then
    rm "$providercomponent_generated_file-tmp" 2>/dev/null
  fi

  echo "Done generating $providercomponent_generated_file"

  cat "$machine_template_file" \
    > "$machine_generated_file"
  echo "Done generating $machine_generated_file"

  cat "$cluster_template_file" \
    > "$cluster_generated_file"
  echo "Done generating $cluster_generated_file"

  echo "You will still need to _edit_ the cluster.yaml and machines.yaml manifests! See the README.md for details."
}

main()
{
  SCRIPT=$(basename "$0")
  OS_TYPE=${OS_TYPE:-ubuntu}
  BASEDIR="$(runpath)"
  OUTPUT_DIR="$BASEDIR/out"
  KUBELET_VERSION=${KUBELET_VERSION:-1.10.6}
  OVERWRITE=0
  SDS_CASE=${SDS_CASE:-0}

  while test $# -gt 0; do
    case "$1" in
      -h|--help)
  # shellcheck disable=SC2140
        echo """
  $SCRIPT - generates input yaml files for Cluster API on openstack. Some environment
        variables are needed set for $SCRIPT to properly function:

        CLUSTER_PRIVATE_KEY   : base64 encoded private key used when make_cluster was run.
        OS_TYPE               : One of "ubuntu" or "centos" -- defaults to "ubuntu"
        CLUSTER_PASSPHRASE    : Only used if CLUSTER_PRIVATE_KEY was generated using a passphrase.
        SDS_CASE              : 0 or 1 -- use 0 if creating a cluster in AWS or MaaS. Use 1 if
                                creating a cluster in SDS Cloud -- defaults to 0.
        KUBELET_VERSION       : e.g. 1.10.6 -- do not prepend a 'v' in front of it -- currently defaults to 1.10.6

  $SCRIPT [options]

  options:
  -h, --help                show brief help
  -f, --force-overwrite     if file to be generated already exists, force script to overwrite it

  Completed manifests are placed in '$OUTPUT_DIR'

        """
        exit 0
        ;;
      -f|--force-overwrite)
        OVERWRITE=1
        shift
        ;;
      *)
        break
        ;;
    esac
  done

  # TODO Fill out the generation pieces as we need them.

  if [[ -z ${CLUSTER_PRIVATE_KEY+x} ]]; then
      echo "Please generate a valid base64 encoded cluster private key and export the key file contents to CLUSTER_PRIVATE_KEY."
      exit 1
  fi

  if [[ -z "${CLUSTER_PASSPHRASE+x}" ]]; then
      echo "Using empty cluster pass phrase to private key"
      CLUSTER_PASSPHRASE='""'
  fi

  if [[ "${OS_TYPE}" =~ (centos|ubuntu) ]]; then
    echo "OS Type set for valid type: $OS_TYPE."
  else
    echo >&2 "Invalid parameter for \$OS_TYPE: '$OS_TYPE'. Must be either 'ubuntu' or 'centos'"
    exit 15
  fi

  if [[ "${SDS_CASE}" =~ [01] ]]; then
    case ${SDS_CASE} in
      0) echo "Creating templates for non-SDS Cloud environment.";;
      1) echo "Creating templates for SDS Cloud environment.";;
      *) echo "Invalid option: for SDS_CASE: ${SDS_CASE}";
        exit 30
      ;;
    esac
  else
    echo >&2 "Invalid parameter for \$SDS_CASE: '$SDS_CASE'. Must be either '1' or '0'."
    exit 20
  fi

  if ! generate_yaml; then
    exit $?
  fi
}

main "$@"
