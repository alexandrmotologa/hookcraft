BINARY_NAME=hookcraft
VERSION=1.0.0
BUILD_DATE=$(shell date +%Y-%m-%d)
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: all build build-web test clean install lint

all: build

build-web:
	@echo "Building React 19 web distribution..."
	cd web && npm install && npm run build

build: build-web
	@echo "Compiling HookCraft single binary..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/hookcraft

test:
	@echo "Running unit test suite..."
	go test -v -race ./...

lint:
	@echo "Running go vet..."
	go vet ./...

clean:
	@echo "Cleaning artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -f hookcraft hookcraft.exe

install: build
	@echo "Installing HookCraft to GOPATH/bin..."
	go install $(LDFLAGS) ./cmd/hookcraft
