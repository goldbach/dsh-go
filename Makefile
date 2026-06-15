BIN     := dsh
GOBIN   := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN   := $(shell go env GOPATH)/bin
endif

.PHONY: build test vet install clean

build:
	go build -o $(BIN) ./cmd/dsh

test:
	go test -v ./...

vet:
	go vet ./...

install:
	go install ./cmd/dsh

clean:
	rm -f $(BIN)
