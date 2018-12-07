FROM centos:7 as builder

# setup sudo and other tools (for testing only)
RUN yum update -y \
    && yum install epel-release -y \
    && yum install gettext python-pip jq yum-utils -y \
    && yum clean all
RUN pip install yq

# kubernetes rpms
RUN echo -e '[kubernetes]\n\
name=Kubernetes\n\
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64\n\
enabled=1\n\
gpgcheck=1\n\
repo_gpgcheck=0'\
>> /etc/yum.repos.d/kubernetes.repo

ARG K8S_VERSION_OLD=1.10.6
ARG K8S_VERSION_NEW=1.11.2

RUN mkdir /var/tmp/${K8S_VERSION_OLD} \
  && yumdownloader --assumeyes --resolve --destdir /var/tmp/${K8S_VERSION_OLD} \
    kubectl-${K8S_VERSION_OLD} \
    kubeadm-${K8S_VERSION_OLD} \
    kubelet-${K8S_VERSION_OLD} \
    kubernetes-cni
RUN mkdir /var/tmp/${K8S_VERSION_NEW} \
  && yumdownloader --assumeyes --resolve --destdir /var/tmp/${K8S_VERSION_NEW} \
    kubectl-${K8S_VERSION_NEW} \
    kubeadm-${K8S_VERSION_NEW} \
    kubelet-${K8S_VERSION_NEW} \
    kubernetes-cni

WORKDIR /usr/scripts
COPY bootstrap_scripts bootstrap_scripts
COPY templates templates
COPY generate-yaml.sh .

# Run generate to test things out
# Setup some variables for testing
ENV OS_TYPE=centos

RUN chmod +x /usr/scripts/generate-yaml.sh \
  && /usr/scripts/generate-yaml.sh \
  && mkdir -p /usr/scripts/master \
  && mkdir -p /usr/scripts/node

# get master scripts
RUN echo '#!/usr/bin/env bash' > /usr/scripts/master/startupScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Master"]))' | jq --raw-output -s .[-1].metadata.startupScript | xargs -0 printf '%b' >> /usr/scripts/master/startupScript.sh \
  && chmod +x /usr/scripts/master/startupScript.sh
RUN echo '#!/usr/bin/env bash' > /usr/scripts/master/shutdownScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Master"]))' | jq --raw-output -s .[-1].metadata.shutdownScript | xargs -0 printf '%b' >> /usr/scripts/master/shutdownScript.sh \
  && chmod +x /usr/scripts/master/shutdownScript.sh
RUN echo '#!/usr/bin/env bash' > /usr/scripts/master/upgradeScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Master"]))' | jq --raw-output -s .[-1].metadata.upgradeScript | xargs -0 printf '%b' >> /usr/scripts/master/upgradeScript.sh \
  && chmod +x /usr/scripts/master/upgradeScript.sh

# get node scripts
RUN echo '#!/usr/bin/env bash' > /usr/scripts/node/startupScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Node"]))' | jq --raw-output -s .[-1].metadata.startupScript | xargs -0 printf '%b' >> /usr/scripts/node/startupScript.sh \
  && chmod +x /usr/scripts/node/startupScript.sh
RUN echo '#!/usr/bin/env bash' > /usr/scripts/node/shutdownScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Node"]))' | jq --raw-output -s .[-1].metadata.shutdownScript | xargs -0 printf '%b' >> /usr/scripts/node/shutdownScript.sh \
  && chmod +x /usr/scripts/node/shutdownScript.sh
RUN echo '#!/usr/bin/env bash' > /usr/scripts/node/upgradeScript.sh \
  && cat /usr/scripts/out/provider-components.yaml | yq  -s --raw-output '.[]  | select(.metadata.name == "machine-setup") | .data."machine_setup_configs.yaml"' | yq '.items[] | select(.machineParams.roles | contains(["Node"]))' | jq --raw-output -s .[-1].metadata.upgradeScript | xargs -0 printf '%b' >> /usr/scripts/node/upgradeScript.sh \
  && chmod +x /usr/scripts/node/upgradeScript.sh


FROM centos:7.4.1708 

WORKDIR /resources

RUN yum update -y \
  && yum install -y yum-utils createrepo wget

COPY --from=builder /var/tmp/${K8S_VERSION_OLD} rpms/${K8S_VERSION_OLD}
COPY --from=builder /var/tmp/${K8S_VERSION_NEW} rpms/${K8S_VERSION_NEW}
COPY --from=builder /usr/scripts ./scripts

RUN mkdir -p /resources/yaml \
  && wget --directory-prefix=/resources/yaml https://raw.githubusercontent.com/coreos/flannel/v0.10.0/Documentation/kube-flannel.yml 

RUN createrepo /resources/rpms/${K8S_VERSION_OLD}

RUN echo -e "[local]\n\
name=Local\n\
baseurl=file:///resources/rpms/${K8S_VERSION_OLD}\n\
enabled=1\n\
gpgcheck=0\n\
repo_gpgcheck=0"\
>> /etc/yum.repos.d/local.repo

# fake vars for script validatio
ENV MASTER_IP=changeme
ENV CONTROL_PLANE_VERSION=changeme
ENV POD_CIDR=changeme
ENV CLUSTER_DNS_DOMAIN=cluster.local
ENV SERVICE_CIDR=changeme

