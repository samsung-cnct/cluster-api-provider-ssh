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
      return 1
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
  local MASTER_UPGRADE_SCRIPT MASTER_TEARDOWN_SCRIPT NODE_UPGRADE_SCRIPT NODE_TEARDOWN_SCRIPT
  local MASTER_BOOTSTRAP_SCRIPT NODE_BOOTSTRAP_SCRIPT

  if ! mkdir -p "${OUTPUT_DIR}" 2>/dev/null; then
    echo >&2 "Unable to mkdir $OUTPUT_DIR"
    return 1
  fi

  bootstrap_dir="$BASEDIR/bootstrap_scripts/${OS_TYPE}-${OS_VER}/${WORK_ENV}"

  # --- MACHINES ---
  machine_template_file="$BASEDIR/templates/machines.yaml.template"
  machine_generated_file=${OUTPUT_DIR}/machines.yaml

  # --- CLUSTERS ---
  cluster_template_file="$BASEDIR/templates/cluster.yaml.template"
  cluster_generated_file=${OUTPUT_DIR}/cluster.yaml

  # --- PROVIDER_CONFIG ---
  providercomponent_template_file="$BASEDIR/templates/provider-components.yaml.template"
  providercomponent_generated_file=${OUTPUT_DIR}/provider-components.yaml

  if [[ $OVERWRITE -ne 1 ]] && [[ -f $providercomponent_generated_file ]]; then
    echo >&2 "File $providercomponent_generated_file already exists. Delete it manually before running this script."
    return 1
  fi

  if [[ ! -d "${bootstrap_dir}" ]]; then
    echo >&2 "Looks like the template directory you seek doesn't exist. Perhaps your \$OS_VER is incorrect.
Please make sure '${bootstrap_dir}' exists!
Verify your OS_VER env variable if set."

    return 1
  fi

  # $bootstrap_dir dictates the template. If the template doesn't exist
  # these will error.
  MASTER_BOOTSTRAP_SCRIPT="$(< "${bootstrap_dir}"/master_bootstrap.template)"
  MASTER_TEARDOWN_SCRIPT="$(< "${bootstrap_dir}"/master_teardown.template)"
  MASTER_UPGRADE_SCRIPT="$(< "${bootstrap_dir}"/master_upgrade.template)"

  NODE_BOOTSTRAP_SCRIPT="$(< "${bootstrap_dir}"/node_bootstrap.template)"
  NODE_TEARDOWN_SCRIPT="$(< "${bootstrap_dir}"/node_teardown.template)"
  NODE_UPGRADE_SCRIPT="$(< "${bootstrap_dir}"/node_upgrade.template)"

  # prepend common functions to template script
  FUNCTIONS=$(< "$bootstrap_dir/common_functions.template")

  export MASTER_BOOTSTRAP_SCRIPT \
         NODE_BOOTSTRAP_SCRIPT MASTER_TEARDOWN_SCRIPT NODE_TEARDOWN_SCRIPT MASTER_UPGRADE_SCRIPT \
         NODE_UPGRADE_SCRIPT FUNCTIONS OS_TYPE KUBELET_VERSION

  # shellcheck disable=SC2016
  envsubst '$MASTER_BOOTSTRAP_SCRIPT
            $NODE_BOOTSTRAP_SCRIPT  $MASTER_TEARDOWN_SCRIPT $NODE_TEARDOWN_SCRIPT
            $MASTER_UPGRADE_SCRIPT $NODE_UPGRADE_SCRIPT $KUBELET_VERSION $OS_TYPE' \
           < "$providercomponent_template_file" > "$providercomponent_generated_file-tmp"

  if inject_functions "$providercomponent_generated_file-tmp" > "$providercomponent_generated_file"; then
    rm "$providercomponent_generated_file-tmp" 2>/dev/null
  fi

  echo "Done generating $providercomponent_generated_file"

  envsubst < "$machine_template_file" > "$machine_generated_file"
  echo "Done generating $machine_generated_file"

  envsubst < "$cluster_template_file" > "$cluster_generated_file"
  echo "Done generating $cluster_generated_file"

  echo "You will still need to _edit_ the cluster.yaml and machines.yaml manifests! See the README.md for details."
}

main()
{
  SCRIPT=$(basename "$0")
  OS_TYPE=${OS_TYPE:-centos}
  OS_VER=${OS_VER:-7.4}
  BASEDIR="$(runpath)"
  OUTPUT_DIR="$BASEDIR/out"
  KUBELET_VERSION=${KUBELET_VERSION:-1.10.6}
  SDS_ENV="${SDS_ENV:-true}"
  OVERWRITE=0

  while test $# -gt 0; do
    case "$1" in
      -h|--help)
  # shellcheck disable=SC2140
        echo """
  $SCRIPT - generates input yaml files for Cluster API on openstack. Some environment
        variables are needed set for $SCRIPT to properly function:

        OS_TYPE               : One of "ubuntu" or "centos" -- defaults to "centos"
        KUBELET_VERSION       : e.g. 1.10.6 -- do not prepend a 'v' in front of it -- currently defaults to 1.10.6
        SDS_ENV               : Create provider-components.yaml for SDS or not? e.g. 0 (non-SDS) or 1 (SDS) -- defaults to 1
        OS_VER                : The distribution version you desire to use. The default is 7.4

  $SCRIPT [options]

  options:
  -h, --help                show brief help
  -f, --force-overwrite     if file to be generated already exists, force script to overwrite it

  Completed manifests are placed in '$OUTPUT_DIR'

        """
        return 0
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
  if [[ "${OS_TYPE}" =~ (centos|ubuntu) ]]; then
    echo "OS Type set for valid type: $OS_TYPE."
  else
    echo >&2 "Invalid parameter for \$OS_TYPE: '$OS_TYPE'. Must be either 'ubuntu' or 'centos'"
    return 1
  fi

  if [[ "${SDS_ENV}" =~ true|false ]]; then
    if [[ "${SDS_ENV}" == true ]]; then
      echo "Setting environment for SDS!!!"
      WORK_ENV="sds"
    else
      echo "Setting environment for AWS!!!"
      WORK_ENV="aws"
    fi
  else
    echo >&2 "Invalid parameter for \$SDS_ENV: '$SDS_ENV'. Must be either 'true' or 'false'"
    return 1
  fi

  if ! generate_yaml; then
    return 1
  fi
}

if ! main "$@"; then
  exit 17
fi
