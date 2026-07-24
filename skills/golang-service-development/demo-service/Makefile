.PHONY: all build run test lint fmt vet generate proto migrate tidy regenerate

## build: Build the server binary
build:
	go build -o bin/server ./cmd/server/

## run: Run the server locally
run:
	go run ./cmd/server/

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

## migrate: Run GORM AutoMigrate
migrate:
	go run ./cmd/migrate/

## tidy: Run go mod tidy
tidy:
	go mod tidy

## regenerate: Re-run all codegen + tidy (call after proto/model changes or template updates)
regenerate: proto generate tidy

## all: Format, vet, lint, test
all: fmt vet lint test
