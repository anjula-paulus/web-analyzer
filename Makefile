PHONY: build test run docker-build docker-run clean

BINARY_NAME=web-analyzer

build:
	go build -o build/$(BINARY_NAME) ./cmd/web-analyzer

build_windows:
	GOOS=windows GOARCH=amd64 go build -o build/$(BINARY_NAME).exe ./cmd/web-analyzer

build_linux:
	GOOS=linux GOARCH=amd64 go build -o build/$(BINARY_NAME)-linux ./cmd/web-analyzer

build_darwin:
	GOOS=darwin GOARCH=amd64 go build -o build/$(BINARY_NAME)-darwin ./cmd/web-analyzer

build_and_run: build
	./build/$(BINARY_NAME)

test:
	go test -v ./...

run:
	go run ./cmd/web-analyzer

# Run with custom config
run_dev:
	CONFIG_PATH=config.yaml go run ./cmd/web-analyzer

# Run with environment variables
run_env:
	PORT=:9090 MAX_WORKERS=20 go run ./cmd/web-analyzer

docker_build:
	docker build -t $(BINARY_NAME) .

docker_run: docker_build
	docker run -p 8080:8080 $(BINARY_NAME)

# Run docker with custom port
docker_run_custom:
	docker run -p 9090:9090 -e PORT=:9090 $(BINARY_NAME)

clean:
	rm -rf build/
	docker rmi $(BINARY_NAME) 2>/dev/null || true