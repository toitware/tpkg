
# The toitdoc page used to server package toitdocs.
WEB_TOITDOCS_VERSION ?= v0.2.10
# The SDK used to extract toitdocs from packages.
SDK_VERSION ?= v2.0.0-alpha.147
# The version of the pkg.toit.io web page.
WEB_TPKG_VERSION ?= v0.2.2

BUILD_DIR := build
PROTO_DIR := proto

GOOGLE_API_PROTO_DIR := third_party/googleapis

PROTO_FLAGS := -I$(dir $(shell which protoc))$(dir $(shell readlink "$(shell which protoc)"))../include/ -I/snap/protobuf/current/include/
PROTO_SOURCES := $(foreach dir,$(PROTO_DIR),$(shell find $(dir) -name '*.proto'))
GO_PROTO_FILES := $(PROTO_SOURCES:%.proto=$(BUILD_DIR)/%.pb.go)
GO_PROTO_PLUGINS := --plugin=protoc-gen-go=$(shell which protoc-gen-go) --plugin=protoc-gen-go-grpc=$(shell which protoc-gen-go-grpc) --plugin=protoc-gen-grpc-gateway=$(shell which protoc-gen-grpc-gateway) --plugin=protoc-gen-openapiv2=$(shell which protoc-gen-openapiv2)
GO_PROTO_FLAGS := $(PROTO_FLAGS) -I$(GOOGLE_API_PROTO_DIR)
GO_PACKAGE := github.com/toitware/tpkg

.PHONY: all
all: registry

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: go_dependencies
go_dependencies:
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install github.com/golang/mock/mockgen
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
	go install github.com/jstroem/tedi/cmd/tedi

build/proto/%.pb.go: proto/%.proto
	@mkdir -p $(BUILD_DIR)/proto/
	protoc $(<:proto/%=%) \
		--go_out=paths=source_relative:$(BUILD_DIR)/proto/ \
		--go-grpc_out=paths=source_relative:$(BUILD_DIR)/proto/ \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt paths=source_relative \
		--grpc-gateway_out=$(BUILD_DIR)/proto/ \
		--proto_path ./proto/ \
		$(GO_PROTO_FLAGS)

.PHONY: protobuf
protobuf: $(GO_PROTO_FILES)

GO_SOURCES := $(shell find . -name '*.go' -not -name '*_mock.go' -not -path './tests/*')
GO_DEPS = $(GO_PROTO_FILES)

GO_BUILD_FLAGS ?=
ifeq ("$(GO_BUILD_FLAGS)", "")
$(eval GO_BUILD_FLAGS=CGO_ENABLED=1 GODEBUG=netdns=go)
else
$(eval GO_BUILD_FLAGS=$(GO_BUILD_FLAGS) CGO_ENABLED=1 GODEBUG=netdns=go)
endif

$(BUILD_DIR)/registry: $(GO_DEPS) $(GO_SOURCES)
	$(GO_BUILD_FLAGS) go build -ldflags "$(GO_LINK_FLAGS)" -tags 'netgo osusergo' -o $(BUILD_DIR)/registry .

.PHONY: registry
registry: $(BUILD_DIR)/registry

$(BUILD_DIR)/registry_container: $(GO_DEPS) $(GO_SOURCES)
	GOOS=linux $(GO_BUILD_FLAGS) go build -ldflags "$(GO_LINK_FLAGS)" -tags 'netgo osusergo' -o $(BUILD_DIR)/registry_container .

GO_MOCKS := controllers/registry_mock.go \
            controllers/toitdoc_mock.go

$(GO_MOCKS): $(GO_DEPS)

%_mock.go: %.go
	mockgen -destination=$@ -source=$< -package=$(notdir $(patsubst %/,%,$(dir $@))) -self_package=$(GO_PACKAGE)/$(patsubst %/,%,$(dir $@))

.PHONY: mocks
mocks:
	$(MAKE) -j$(getconf _NPROCESSORS_ONLN) $(GO_MOCKS)

TEST_FLAGS ?=
.PHONY: test
test: $(GO_MOCKS)
	tedi test -v -cover $(TEST_FLAGS) $(foreach dir,$(filter-out third_party/, $(sort $(dir $(wildcard */)))),./$(dir)...)

$(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION):
	mkdir -p $(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION)
	curl -L -o $(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION)/build.tar.gz \
		https://github.com/toitware/web-toitdocs/releases/download/$(WEB_TOITDOCS_VERSION)/build.tar.gz
	cd $(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION) && tar -xzf build.tar.gz
	rm -rf $(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION)/build.tar.gz

$(BUILD_DIR)/sdk/$(SDK_VERSION):
	mkdir -p $(BUILD_DIR)/sdk/$(SDK_VERSION)
	curl -L -o $(BUILD_DIR)/sdk/$(SDK_VERSION)/toit-linux.tar.gz \
	  https://github.com/toitlang/toit/releases/download/$(SDK_VERSION)/toit-linux.tar.gz
	cd $(BUILD_DIR)/sdk/$(SDK_VERSION) && tar --strip-components=1 -xzf toit-linux.tar.gz
	rm -rf $(BUILD_DIR)/sdk/$(SDK_VERSION)/toit-linux.tar.gz

$(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION):
	mkdir -p $(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION)
	curl -L -o $(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION)/build.tgz \
	  https://github.com/toitware/web-tpkg/releases/download/$(WEB_TPKG_VERSION)/build.tgz
	cd $(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION) && tar -xzf build.tgz
	rm -rf $(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION)/build.tgz

TOITC_PATH ?= `pwd`/../toit/build/host/sdk/bin/toit.compile
TOITLSP_PATH ?= `pwd`/../toit/build/host/sdk/bin/toit.lsp
SDK_PATH ?=`pwd`/../toit/

.PHONY: run/registry
run/registry: $(BUILD_DIR)/registry
	rm -rf /tmp/toitdocs
	rm -rf /tmp/registry
	TOITC_PATH=$(TOITC_PATH) TOITLSP_PATH=$(TOITLSP_PATH) SDK_PATH=$(SDK_PATH) ./$(BUILD_DIR)/registry

.PHONY: image-dependencies
image-dependencies: $(BUILD_DIR)/registry_container $(BUILD_DIR)/web_toitdocs/$(WEB_TOITDOCS_VERSION) $(BUILD_DIR)/sdk/$(SDK_VERSION) $(BUILD_DIR)/web_tpkg/$(WEB_TPKG_VERSION)

.PHONY: image
image: image-dependencies
	docker build --build-arg WEB_TOITDOCS_VERSION=$(WEB_TOITDOCS_VERSION) --build-arg SDK_VERSION=${SDK_VERSION} --build-arg WEB_TPKG_VERSION=${WEB_TPKG_VERSION} -t toit_registry .

GCLOUD_IMAGE_TAG ?= $(USER)
.PHONY: gcloud
gcloud: image
	docker tag toit_registry:latest gcr.io/infrastructure-220307/toit_registry:$(subst +,-,$(GCLOUD_IMAGE_TAG))
	docker push gcr.io/infrastructure-220307/toit_registry:$(subst +,-,$(GCLOUD_IMAGE_TAG))

.PHONY: docker-build-args
docker-build-args:
	@echo WEB_TOITDOCS_VERSION=$(WEB_TOITDOCS_VERSION)
	@echo SDK_VERSION=$(SDK_VERSION)
	@echo WEB_TPKG_VERSION=$(WEB_TPKG_VERSION)

.PHONY: clean
clean:
	rm -rf ./$(BUILD_DIR) $(GO_MOCKS)
