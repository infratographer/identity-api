all: lint test
PHONY: deps generate test coverage lint golint clean vendor docker-up docker-down unit-test
GOOS=linux
APP_NAME?=identity-api

DEV_DB="identity_api_dev"

TEST_PRIVKEY_FILE?=tests/data/privkey.pem
CONFIG_FILE?=identity-api.example.yaml

.PHONY: help
help: Makefile ## Print help
	@grep -h "##" $(MAKEFILE_LIST) | grep -v grep | sed -e 's/:.*##/#/' | column -c 2 -t -s#

# we use a prerelease version of oapi-codegen because the latest version (v1.12.4)
# produces buggy Gin code
deps:
	@go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@f4cf8f9

generate: deps ## run openapi code generation
	@go generate ./...

test: | unit-test ## run linting and unit tests

unit-test: | lint 
	@echo Running unit tests...
	@go test -cover -short -tags testtools ./...

coverage:
	@echo Generating coverage report...
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint ## run the linter

golint: | vendor
	@echo Linting Go files...
	@golangci-lint run --build-tags "-tags testtools"

build: vendor generate ## build the app binary
	@go mod download
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o bin/${APP_NAME}

clean: docker-clean ## clean up docker and the build
	@echo Cleaning...
	@rm -f app
	@rm -rf ./dist/
	@rm -rf coverage.out
	@go clean -testcache

vendor:
	@go mod download
	@go mod tidy

.PHONY: run
up: build $(TEST_PRIVKEY_FILE) ## start the app
	bin/${APP_NAME} serve --config ${CONFIG_FILE}

$(TEST_PRIVKEY_FILE):
	openssl genpkey -out $(TEST_PRIVKEY_FILE) -algorithm RSA -pkeyopt rsa_keygen_bits:4096

dev-database: | build ## create and migrate the dev database
	@echo --- Creating dev database...
	@date --rfc-3339=seconds
	@cockroach sql -e "drop database if exists ${DEV_DB}"
	@cockroach sql -e "create database ${DEV_DB}"
	@bin/${APP_NAME} migrate --config=${CONFIG_FILE}
