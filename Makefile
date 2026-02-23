.PHONY: build run test clean install lint fmt vet

BINARY_NAME=pulsego
GO=go
MAIN_PATH=cmd/pulsego

build:
	$(GO) build -o $(BINARY_NAME) ./$(MAIN_PATH)

run:
	$(GO) run ./$(MAIN_PATH)

test:
	$(GO) test -v ./...

clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

install:
	$(GO) install ./...

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

all: clean build
