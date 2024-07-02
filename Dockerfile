# TODO(anders): Use i386/busybox:glibc, but currently we need multiarch++ due to pyinstaller.
FROM ubuntu:22.04

RUN \
  dpkg --add-architecture i386 && \
  apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y libc6:i386 libncurses5:i386 libstdc++6:i386 zlib1g:i386 ca-certificates ssh && \
  rm -rf /var/lib/apt/lists/* && \
  update-ca-certificates

WORKDIR /

ENV PORT=8733
ENV DEBUG_PORT=8520

ENV SDK_PATH=/sdk
ENV TOITDOCS_VIEWER_PATH=/web_toitdocs
ENV TPKG_PATH=/web_tpkg

COPY config/config.yaml /config/config.yaml
COPY build/registry_container /registry_container
COPY build/web_toitdocs $TOITDOCS_VIEWER_PATH
COPY build/sdk $SDK_PATH
COPY build/web_tpkg $TPKG_PATH

# Bake in the keys of common git servers.
# Use the ENV variable 'SSH_KNOWN_HOSTS' to replace this file with a custom one.
ENV SSH_KNOWN_HOSTS=/etc/ssh/ssh_known_hosts
RUN mkdir -p /etc/ssh
RUN ssh-keyscan github.com >> /etc/ssh/ssh_known_hosts
RUN ssh-keyscan gitlab.com >> /etc/ssh/ssh_known_hosts
RUN ssh-keyscan shell.sf.net >> /etc/ssh/ssh_known_hosts

ENTRYPOINT ["/registry_container"]
