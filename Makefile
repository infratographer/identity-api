ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR := .tools

# Determine OS and ARCH for some tool versions.
OS := linux
ARCH := amd64

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	OS = darwin
endif

UNAME_P := $(shell uname -p)
ifneq ($(filter arm%,$(UNAME_P)),)
	ARCH = arm64
endif

# Tool Versions
COCKROACH_VERSION = v22.2.8

OS_VERSION = $(OS)
ifeq ($(OS),darwin)
OS_VERSION = darwin-10.9
ifeq ($(ARCH),arm64)
OS_VERSION = darwin-11.0
endif
endif

COCKROACH_VERSION_FILE = cockroach-$(COCKROACH_VERSION).$(OS_VERSION)-$(ARCH)
COCKROACH_RELEASE_URL = https://binaries.cockroachdb.com/$(COCKROACH_VERSION_FILE).tgz

# go files to be checked
GO_FILES=$(shell git ls-files '*.go')

GOOS=linux
APP_NAME?=identity-api

DEV_DB=identity_api_dev
DEV_URI="postgresql://root@cockroachdb:26257/${DEV_DB}?sslmode=disable"

TEST_PRIVKEY_FILE?=tests/data/privkey.pem
CONFIG_FILE?=identity-api.example.yaml

DOCKER_BUILD_TAG?=latest

# targets

.PHONY: help all generate dev-database test unit-test coverage lint golint build clean vendor up

help: Makefile ## Print help.
	@grep -h "##" $(MAKEFILE_LIST) | grep -v grep | sed -e 's/:.*##/#/' | column -c 2 -t -s#

all: lint test  ## Runs lint checks and tests.

generate:  ## Runs all code generation.
	@go generate ./...

dev-database: | vendor $(TOOLS_DIR)/cockroach  ## Initializes dev database "${DEV_DB}"
	@$(TOOLS_DIR)/cockroach sql -e "drop database if exists ${DEV_DB}"
	@$(TOOLS_DIR)/cockroach sql -e "create database ${DEV_DB}"
	@IDAPI_CRDB_URI="${DEV_URI}" \
		go run main.go migrate up

test: | unit-test  ## Run unit tests.

unit-test: | generate $(TOOLS_DIR)/cockroach  ## Runs unit tests.
	@echo Running unit tests...
	@COCKROACH_BINARY="$(ROOT_DIR)/$(TOOLS_DIR)/cockroach" \
		go test -race -cover -short ./...

coverage: | $(TOOLS_DIR)/cockroach  ## Runs tests generating coverage output.
	@echo Generating coverage report...
	@export COCKROACH_BINARY="$(ROOT_DIR)/$(TOOLS_DIR)/cockroach" && \
		go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1 && \
		go tool cover -func=coverage.out && \
		go tool cover -html=coverage.out

lint: golint  ## Runs go lint checks.

golint: | vendor  ## Runs go lint checks.
	@echo Linting Go files...
	@go tool golangci-lint run --build-tags "-tags testtools"

fixlint:
	@echo Fixing go imports
	@find . -type f -iname '*.go' | xargs go tool goimports -w -local go.infratographer.com/identity-api

build: vendor generate  ## Builds a binary stored at bin/${APP_NAME}
	@echo Building image...
	@CGO_ENABLED=0 go build -buildvcs=false -mod=readonly -v -o bin/${APP_NAME}

docker: build  ## Builds a docker image tagged with $(APP_NAME):$(DOCKER_BUILD_TAG)
	@echo Building docker image...
	@docker build --file Dockerfile -t $(APP_NAME):$(DOCKER_BUILD_TAG) bin/

clean:  ## Cleans up generated files.
	@echo Cleaning...
	@rm -f app
	@rm -rf ./dist/
	@rm -rf coverage.out
	@rm -rf .tools/
	@go clean -testcache

vendor:  ## Downloads and tidies go modules.
	@go mod download
	@go mod tidy

up: build $(TEST_PRIVKEY_FILE)  ## Builds and runs the service.
	bin/${APP_NAME} serve --config ${CONFIG_FILE}

$(TEST_PRIVKEY_FILE):
	openssl genpkey -out $(TEST_PRIVKEY_FILE) -algorithm RSA -pkeyopt rsa_keygen_bits:4096

# Tools setup
$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

$(TOOLS_DIR)/cockroach: | $(TOOLS_DIR)
	@echo "Downloading cockroach: $(COCKROACH_RELEASE_URL)"
	@curl --silent --fail "$(COCKROACH_RELEASE_URL)" \
		| tar -xz --strip-components 1 -C $(TOOLS_DIR) $(COCKROACH_VERSION_FILE)/cockroach

	$@ version
