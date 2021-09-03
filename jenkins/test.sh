#!/bin/bash
set -ex

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
TPKG_DIR="$(cd "$CURR_DIR/../" && pwd)"
TOOLS_DIR="$TPKG_DIR/tools"

make -j 10 tpkg
go get -u github.com/jstemmer/go-junit-report

GROUP_TAG="test"
if [ "$BUILD_TAG" != "" ]; then
  GROUP_TAG="test-$EXECUTOR_NUMBER"
fi

# Set up variables for test execution
export TPKG_CMD_PATH=$TOIT_TPKG
export TOIT_SDK_PATH=$TPKG_DIR/sdk.tgz

# Setup config
if [ "$TOIT_FIRMWARE_VERSION" != "" ]; then
  mkdir "$TOOLS_DIR"
  pushd "$TOOLS_DIR"
  gsutil cp gs://toit-binaries/$TOIT_FIRMWARE_VERSION/sdk/$TOIT_FIRMWARE_VERSION.tar .
  #gsutil cp gs://toit-archive/toit-devkit/linux/$TOIT_FIRMWARE_VERSION.tgz $TOIT_SDK_PATH
  tar x -fv $TOIT_FIRMWARE_VERSION.tar
  popd
fi

export TPKG_PATH="$TPKG_DIR/build/tpkg"
export TOITLSP_PATH="$TOOLS_DIR/toitlsp"
export TOITC_PATH="$TOOLS_DIR/toitc"
export TOITVM_PATH="$TOOLS_DIR/toitvm"
GROUP_TAG=$GROUP_TAG tedi test -v -cover -bench=. ./tests/... 2>&1 | tee tests.out
cat tests.out | go-junit-report > tests.xml
