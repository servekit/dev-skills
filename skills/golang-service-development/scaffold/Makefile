.PHONY: all build run clean

## build: Build the scaffold generator binary to bin/scaffold
build:
	go build -o bin/scaffold .

## run: Run the scaffold generator (examples: make run -- demo; make run -- --force demo ../skills/golang-service-development)
run:
	go run . $(ARGS)

## clean: Remove build artifacts
clean:
	rm -rf bin/
