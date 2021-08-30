FROM gcr.io/infrastructure-220307/console-base-ia32:latest

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

ENTRYPOINT ["/registry_container"]
