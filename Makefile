.PHONY: dir

GO ?= go
GOBUILDFLAGS ?= -ldflags="-s -w"
bd = bin
exe = fio-docker-fsck
linter := $(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)
rev = $(shell git rev-parse --short HEAD)

all: $(exe)

$(bd):
	@mkdir -p $@

$(exe): $(bd) main.go
	$(GO) build $(GOBUILDFLAGS) -o $(bd)/$@

clean:
	@rm -r $(bd)

format:
	@gofmt -l -w ./

check:
	@test -z $(shell gofmt -l ./) || echo "[WARN] Fix formatting issues with 'make format'"
	$(linter) run

test: $(exe)
	go test -v
