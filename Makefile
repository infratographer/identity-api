all: lint test
PHONY: test coverage lint golint clean vendor docker-up docker-down unit-test
GOOS=linux
# use the working dir as the app name, this should be the repo name
APP_NAME=$(shell basename $(CURDIR))

TEST_PRIVKEY_FILE?=tests/data/privkey.pem

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

build:
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

docker-up: $(TEST_PRIVKEY_FILE)
	@docker-compose build
	@docker-compose  -f docker-compose.yml up -d app

docker-down:
	@docker-compose -f docker-compose.yml down

docker-clean:
	@docker-compose -f docker-compose.yml down --volumes

$(TEST_PRIVKEY_FILE):
	openssl genpkey -out $(TEST_PRIVKEY_FILE) -algorithm RSA -pkeyopt rsa_keygen_bits:4096
