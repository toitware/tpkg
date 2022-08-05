# TODO(anders): Use i386/busybox:glibc, but currently we need multiarch++ due to pyinstaller.
FROM ubuntu:22.04

RUN \
  dpkg --add-architecture i386 && \
  apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y libc6:i386 libncurses5:i386 libstdc++6:i386 zlib1g:i386 ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  update-ca-certificates

WORKDIR /

ENV PORT 8733
ENV DEBUG_PORT 8520

ARG WEB_TOITDOCS_VERSION
ARG SDK_VERSION
ARG WEB_TPKG_VERSION

COPY config/config.yaml /config/config.yaml
COPY build/registry_container /registry_container
copy build/web_toitdocs/$WEB_TOITDOCS_VERSION /web_toitdocs
copy build/sdk/$SDK_VERSION /sdk
copy build/web_tpkg/$WEB_TPKG_VERSION /web_tpkg

ENV SDK_PATH /sdk
ENV TOITDOCS_VIEWER_PATH /web_toitdocs
ENV TPKG_PATH /web_tpkg

ENV SDK_PATH /sdk
ENV TOITDOCS_VIEWER_PATH /web_toitdocs

ENV SDK_PATH /sdk
ENV TOITDOCS_VIEWER_PATH /web_toitdocs

# We are baking in private data.
# As of 2022-07-08 the deployment overrides these values:
# https://github.com/toitware/deployment/blob/50d35c2498cb98f360c922a491c2c31e73cc403d/console/values.yaml#L437
# However, by adding the key here, we can remove these lines from there.

# When building locally, one can either get the real key from bitwarden, or use any key.
# It should only be necessary when pushing to the registry. (Not 100% certain.)
copy private_ssh_key /ssh_data/private_ssh_key
ENV REGISTRY_SSH_KEY_FILE /ssh_data/private_ssh_key

# Same: we are baking in the known_hosts, which is, as of 2022-07-08, overridden by the deployment.
copy known_hosts /ssh_data/known_hosts
ENV SSH_KNOWN_HOSTS /ssh_data/known_hosts

ENTRYPOINT ["/registry_container"]
