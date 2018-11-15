FROM centos:7 as builder

# setup sudo and other tools (for testing only)
ENV SUDO_USER=docker
ENV SUDO_USER_PASS=password
RUN yum update -y \
    && yum install epel-release -y \
    && yum install sudo gettext python-pip jq yum-utils -y \
    && useradd -g wheel ${SUDO_USER} \
    && echo "${SUDO_USER}:${SUDO_USER_PASS}" | chpasswd \
    && sed -i -e 's/^\(%wheel\s\+.\+\)/#\1/gi' /etc/sudoers \
    && echo -e '\n%wheel ALL=(ALL) ALL' >> /etc/sudoers \
    && echo -e '\nDefaults:root   !requiretty' >> /etc/sudoers \
    && echo -e '\nDefaults:%wheel !requiretty' >> /etc/sudoers \
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

ENV CLUSTER_PRIVATE_KEY="LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tTUlJRXZnSUJBREFOQmdrcWhraUc5dzBCQVFFRkFBU0NCS2d3Z2dTa0FnRUFBb0lCQVFEVFRQRndTaVZiUU1pQUR2N0h6NGZkTlRkVVdPTS81ZWs1aGFHOTlpcWUyRDZEUXFXNmtXVlJrRmtFb0JxMFdkSHEyamZjdUNnRnhFRHpFdmlIdGlQWDhaR1g1OVhYYWNRclVlVXRVd1ZyaVhmMDg2QTZsM2dyRjhOT1ZPV2tNV1lhZFpvc1BORlF3YWY0Tzg5ZjRyVHVzYWNkelhIL09FUjYySkFZOTQ2NGNpM0xYZnlqckZJUUdBRlliMFl5eEEvbFdVS3ZCSUNOaDhUTlhVOFhBeXpJTTMwL1lRZm1BM1ZicUhjSWdMclBTV1FPNktLOWY5VnlUWDQ3MDU3NTNsTElZYzRKZ0lZR0w3VFVJNTRjMjcycTJjZjZyV0dJWDFQdGZWelZ2M2ZkVHFQeWpBd0JDRDNaMnpRTGFBMVAyZ0NSYnBBYjQrZkJIbXJKSWhUN1VjS2hBZ01CQUFFQ2dnRUFXNzR4cWthWE42ckdJMHZmZFpQMjhxWFRjeXp3NW1LRncvTHdpaEVpNmpzTDBXUVRueERZQS9UL29aRitac1lFN2NwM3JMN01xY3hQUXZyK3E3WDZsQjBQMFB2WkRpR2ZDZ3dSekg4YWdGbDhtdjlZbDNnQ3Rmc29VbzlyOGNGOUNvbjdPYS9pVmdUdlVFbHZ2YnNxUWVVaW9UNzlld25NT2VaamttT1NOU1NibVF4UCt3SWNTcEh2NkxWbnVVS0NabC9VT216QzdsRC9kb2txVks2cXp5eUFjbDdtOG5NRjdUYUpnNW9DelpvckNnVXc2d1J6eGdOYVpiZEQvNHBIaXl1YnVsZHBqWFIzaHRjT24xY3ZXV09QR3gxSVdoaDEzVGtlVWxMOGY1L0Roc2VmbnNoVm9uZytyYWZNZnFuK0tyWmZsektnYzRmeWwvZ1Bqc0lMRFFLQmdRRDZSeUJlK3hwQmsrS2ZDT0JFd3lTSkU3Z3U0Nkt4UXZSRkVaeVl0c2s1RWI4cERZWDdrVjN5bWIxQSs1a1BLZUFSSG1OYWJ0WDFCbmtUbFZ3NExtMXNYd0RpSFdwNUZCQzkwYlVGS1JHSnRqRCtuNWtPZmtjaXVPSVVxaHM4anl0SFI1SS9yRytqbW5BTXBxT1hhbnF3ZU1rVGY4dWZROGt5ZHJWL2VseFhZd0tCZ1FEWUlhN2RsZnVPZWpEMEQxUWdxRThieVdDUTJ0SUg2QmNoajRCbGYwMEZQNmFVSXBwcHpZUldNZVZraXoySDd0REF1VjU0cmpJZnZFdDA3SC9jYU9nMmlDZWh1aW1UQStYdmFkN1JVbUhxbkJqVjVCendZTzZWUUUrMnpHWnRHdkhjYW1Nc0puWXRjKzdrQW5CSlVvWkFmNnZjMTRaY1IrT2VPQXBCTnpVbkt3S0JnUURXR0E0SzhnYU4zbkhYMVlIemtITXFoNkhoQ24xYjRZaVBwUGN1Yy81Q1FYb2pKaHhkRFFjdXhuRHNxODFNK1dkaGZYdEVHdGdRR0k5dVB6QjZKczBZRFZUMkdHL0NUM1hheXc5czhLcStBUXA4Wmc1S0paKzRoaXR2dmV3NnE4WTczVyt2NUZSRHdaTGtYNUJGc0tKdlg0T3ltSG1aTGlWb2M3L2k0c3NuUFFLQmdRQzFsVThLSWhsd01kN3hIZlBMWUR6ZWxva0lDcUJOUGtRNDBRekp1WE5uUm52WWwxM3VLRm5YNVNIYUlGWDQzNzhKUEhSUHBlSzg0UUR3WWtabVVzbE9ENFJranppZ3FMVEZPY2lYU3NTelRLUmRCOEw4ZmlQVXgyeHhvemNtMFcyRWh1bDlUVzhhYnYyMUdRdG52dXJHUTcyTDFKT3c4a0VkUyszcHJzMThjd0tCZ0ZuUE1YOUQzZVdZT2Q3VERoU1NJTitWZFBhMkgvZjIwNWphU081dTdobGlBcFBHQ0xXQVF4Vk9jK040d0h2SGRBNlBVZUhPVllhR0V3eXJKY05uMS9ZNTduWXkzWjJxd0xVNkFxYW5sUXhMSkVVd3lhcWdWODVoYTYwY3NGbldpMTk0NzBzRVRFVTN3YkdJU3BheFpRV3pvOE1tVnAyUTEvVUROOVpFdEE2Sy0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0="
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
