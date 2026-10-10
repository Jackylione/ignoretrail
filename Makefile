export GOCACHE := $(CURDIR)/.cache/go-build
export GOPATH := $(CURDIR)/.cache/go
export GOTMPDIR := $(CURDIR)/.tmp
export TMPDIR := $(CURDIR)/.tmp
export GOTOOLCHAIN := local
export GOENV := off
export GOPROXY := off
export GOSUMDB := off
export GOFLAGS := -mod=vendor
export XDG_CONFIG_HOME := $(CURDIR)/.cache/config
export TEST_TELEMETRY_DIR := $(CURDIR)/.cache/telemetry

.PHONY: prepare build test vet

prepare:
	mkdir -p .tmp .cache/config .cache/telemetry
	printf 'off\n' > .cache/telemetry/mode

build: prepare
	mkdir -p bin
	go build -trimpath -o bin/ignoretrail .

test: prepare
	go test -count=1 -v ./...

vet: prepare
	go vet ./...
