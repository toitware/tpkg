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
COPY build/web_toitdocs/$WEB_TOITDOCS_VERSION /web_toitdocs
COPY build/sdk/$SDK_VERSION /sdk
COPY build/web_tpkg/$WEB_TPKG_VERSION /web_tpkg

ENV SDK_PATH=/sdk
ENV TOITDOCS_VIEWER_PATH=/web_toitdocs
ENV TPKG_PATH=/web_tpkg

ENTRYPOINT ["/registry_container"]
