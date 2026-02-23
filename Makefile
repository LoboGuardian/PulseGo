.PHONY: build run test clean install lint fmt vet install-man uninstall-man

BINARY_NAME=pulsego
GO=go
MAIN_PATH=cmd/pulsego
PREFIX?=/usr/local
MAN_DIR=$(PREFIX)/share/man/man1

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

install-man:
	install -d $(DESTDIR)$(MAN_DIR)
	install -m 644 man/man1/pulsego.1 $(DESTDIR)$(MAN_DIR)/
	gzip -f $(DESTDIR)$(MAN_DIR)/pulsego.1

uninstall-man:
	rm -f $(DESTDIR)$(MAN_DIR)/pulsego.1.gz

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

all: clean build
