.PHONY: all build run test lint fmt vet generate proto migrate tidy regenerate

# Published binary name. Override to ship under a different name without
# touching the source tree, e.g. `make build BIN_NAME=msgsvc`.
# The Go package path stays cmd/server regardless.
BIN_NAME := demo-service
CMD_DIR  := cmd/server

## build: Build the $(BIN_NAME) binary (server + migrate in one)
build:
	go build -o bin/$(BIN_NAME) ./$(CMD_DIR)/

## run: Run the server locally
run:
	go run ./$(CMD_DIR)/

## test: Run tests with race detector
test:
	go test -race -coverprofile=coverage.out ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	gofmt -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## generate: Run gorm.io/cli code generation
generate:
	gorm gen -i ./internal/store/models -o ./internal/store/generated

## proto: Generate protobuf code with buf
proto:
	buf generate

## migrate: Run database migrations (AutoMigrate) via the unified binary
migrate:
	go run ./$(CMD_DIR) migrate

## tidy: Run go mod tidy
tidy:
	go mod tidy

## regenerate: Re-run all codegen + tidy (call after proto/model changes or template updates)
regenerate: proto generate tidy

## all: Format, vet, lint, test
all: fmt vet lint test
