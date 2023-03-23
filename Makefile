all: lint test
PHONY: deps generate test coverage lint golint clean vendor docker-up docker-down unit-test
GOOS=linux
APP_NAME?=identity-api

DB=identity_api
DEV_DB="${DB}_dev"

TEST_PRIVKEY_FILE?=tests/data/privkey.pem
CONFIG_FILE?=identity-api.example.yaml

# we use a prerelease version of oapi-codegen because the latest version (v1.12.4)
# produces buggy Gin code
deps:
	@go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@f4cf8f9

generate: deps
	@go generate ./...

test: | unit-test

unit-test: | lint
	@echo Running unit tests...
	@go test -cover -short -tags testtools ./...

coverage:
	@echo Generating coverage report...
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint

golint: | vendor
	@echo Linting Go files...
	@golangci-lint run --build-tags "-tags testtools"

build: vendor generate
	@go mod download
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o bin/${APP_NAME}

clean: docker-clean
	@echo Cleaning...
	@rm -f app
	@rm -rf ./dist/
	@rm -rf coverage.out
	@go clean -testcache

vendor:
	@go mod download
	@go mod tidy

up: build $(TEST_PRIVKEY_FILE)
	bin/${APP_NAME} serve --config ${CONFIG_FILE}

$(TEST_PRIVKEY_FILE):
	openssl genpkey -out $(TEST_PRIVKEY_FILE) -algorithm RSA -pkeyopt rsa_keygen_bits:4096

dev-database: | vendor
	@echo --- Creating dev database...
	@date --rfc-3339=seconds
	@cockroach sql -e "drop database if exists ${DEV_DB}"
	@cockroach sql -e "create database ${DEV_DB}"
	@go run main.go migrate --config=${CONFIG_FILE} up
