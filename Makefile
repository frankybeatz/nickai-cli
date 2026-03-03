VERSION := 0.8.0
BINARY  := nickai
BUILD   := build

LDFLAGS := -s -w -X main.version=$(VERSION)

# Default: build for current platform.
.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Run locally.
.PHONY: run
run:
	go run .

# Run tests.
.PHONY: test
test:
	go vet ./...
	go test ./...

# Install to GOPATH/bin.
.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" .

# Build all release binaries.
.PHONY: release
release: clean
	@mkdir -p $(BUILD)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD)/$(BINARY)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD)/$(BINARY)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD)/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD)/$(BINARY)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD)/$(BINARY)-windows-amd64.exe .
	@echo "Binaries in $(BUILD)/"
	@ls -lh $(BUILD)/

# Build Nick Node binary.
.PHONY: node
node:
	go build -ldflags "$(LDFLAGS)" -o nickai-node ./cmd/node/

# Build Docker image.
.PHONY: docker
docker:
	docker build -t nickai .

# Run Docker container.
.PHONY: docker-run
docker-run: docker
	docker compose up -d

# Clean build artifacts.
.PHONY: clean
clean:
	rm -rf $(BUILD) $(BINARY) nickai-node

# Generate demo GIF (requires vhs: brew install vhs).
.PHONY: demo
demo:
	vhs demo.tape
