.PHONY: all clean test

ifeq ($(GO_CMD),)
GO_CMD=go
endif

VERSION=$(shell git describe --always --tags | sed -e 's/-/+/')
GO_BUILD=CGO_ENABLED=0 $(GO_CMD) build -ldflags "-X main.version=$(VERSION)" -trimpath

SRCS_OTHER=$(shell find . -type d -name vendor -prune -o -type d -name cmd -prune -o -type f -name "*.go" -print) go.mod

DIR_BIN := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))/bin

DIST_EBSCHEDULE=dist/ebschedule

DISTS=\
	$(DIST_EBSCHEDULE)

TARGETS=\
	$(DISTS)

all: $(TARGETS)
	@echo "$@ done." 1>&2

test:
	TZ=UTC $(GO_CMD) test -covermode atomic -cover ./...
	@echo "$@ done." 1>&2

.PHONY: lint
lint: tools
	$(TOOL_STATICCHECK) ./...

TOOL_STATICCHECK = $(DIR_BIN)/staticcheck
TOOL_MOCKGEN = $(DIR_BIN)/mockgen

TOOLS = \
	$(TOOL_MOCKGEN) \
	$(TOOL_STATICCHECK)

TOOLS_DEP = Makefile

.PHONY: tools
tools: $(TOOLS)
	@echo "$@ done." 1>&2

$(TOOL_STATICCHECK): export GOBIN=$(DIR_BIN)
$(TOOL_STATICCHECK): $(TOOLS_DEP)
	@echo "### `basename $@` install destination=$(GOBIN)" 1>&2
	CGO_ENABLED=0 $(GO_CMD) install honnef.co/go/tools/cmd/staticcheck@v0.4.3

$(TOOL_MOCKGEN): export GOBIN=$(DIR_BIN)
$(TOOL_MOCKGEN): $(TOOLS_DEP)
	@echo "### `basename $@` install destination=$(GOBIN)" 1>&2
	CGO_ENABLED=0 $(GO_CMD) install github.com/golang/mock/mockgen@v1.6.0

.PHONY: gen
TMP_PATH := $(DIR_BIN):$(PATH)
gen: export PATH=$(TMP_PATH)
gen: $(TOOL_MOCKGEN)
	$(GO_CMD) generate ./...
	@echo "$@ done." 1>&2

.PHONY: dist
dist: $(DISTS)
	@echo "$@ done." 1>&2

clean: 
	/bin/rm -f $(TARGETS)
	@echo "$@ done." 1>&2

$(DIST_EBSCHEDULE): cmd/ebschedule/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@  ./cmd/ebschedule/
