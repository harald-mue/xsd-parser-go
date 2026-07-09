GO ?= go
PACKAGE ?= model
SCHEMA ?= examples/minimal/schema.xsd
OUT ?= generated/model
TMP_OUT ?= /tmp/xsd-parser-go-generated
FEATURE ?=

.PHONY: all help fmt vet build install test test-internal test-snapshot test-roundtrip update-snapshots inspect generate generate-minimal compile-generated clean

all: fmt test

help:
	@echo "xsd-parser-go Make targets"
	@echo ""
	@echo "  make fmt                 Format Go sources"
	@echo "  make vet                 Run go vet"
	@echo "  make build               Build all packages"
	@echo "  make install             Install xsd-parser-go into GOPATH/bin"
	@echo "  make test                Run full test suite"
	@echo "  make test-internal       Run internal/cmd tests only"
	@echo "  make test-snapshot       Run generated-code snapshot tests"
	@echo "  make test-roundtrip      Run generated XML round-trip tests"
	@echo "  make update-snapshots    Regenerate golden snapshots"
	@echo "  make inspect             Inspect SCHEMA=$(SCHEMA)"
	@echo "  make generate            Generate PACKAGE=$(PACKAGE) OUT=$(OUT) SCHEMA=$(SCHEMA)"
	@echo "  make generate-minimal    Generate examples/minimal/schema.xsd"
	@echo "  make compile-generated   Generate SCHEMA into temp dir and compile it"
	@echo "  make clean               Remove generated output"
	@echo ""
	@echo "Variables:"
	@echo "  SCHEMA=path/to/schema.xsd"
	@echo "  PACKAGE=model"
	@echo "  OUT=generated/model"
	@echo "  FEATURE=name             Optional snapshot subtest for update-snapshots"

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build ./...

install: clean build
	$(GO) install ./cmd/xsd-parser-go

test:
	$(GO) test ./...

test-internal:
	$(GO) test ./cmd/... ./internal/...

test-snapshot:
	$(GO) test ./tests/snapshot

test-roundtrip:
	$(GO) test ./tests/roundtrip

update-snapshots:
	@if [ -n "$(FEATURE)" ]; then \
		$(GO) test ./tests/snapshot -update -run "TestSnapshots/$(FEATURE)"; \
	else \
		$(GO) test ./tests/snapshot -update; \
	fi

inspect:
	$(GO) run ./cmd/xsd-parser-go inspect $(SCHEMA)

generate:
	$(GO) run ./cmd/xsd-parser-go generate --package $(PACKAGE) --out $(OUT) $(SCHEMA)

generate-minimal:
	$(MAKE) generate SCHEMA=examples/minimal/schema.xsd PACKAGE=$(PACKAGE) OUT=$(OUT)

compile-generated:
	rm -rf $(TMP_OUT)
	$(GO) run ./cmd/xsd-parser-go generate --package $(PACKAGE) --out $(TMP_OUT) $(SCHEMA)
	cd $(TMP_OUT) && $(GO) mod init tmp/generated >/dev/null && $(GO) test ./...

clean:
	rm -rf generated
	rm -rf $(TMP_OUT)
