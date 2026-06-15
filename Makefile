BIN     := dsh
GOBIN   := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN   := $(shell go env GOPATH)/bin
endif

.PHONY: build test vet install clean

build:
	go build -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

install:
	go install .

clean:
	rm -f $(BIN)
