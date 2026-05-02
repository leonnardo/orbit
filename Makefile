BIN_DIR ?= $(HOME)/.local/bin
BIN     := $(BIN_DIR)/orbit
PKG     := .

GO ?= go

.PHONY: help build install test smoke tidy fmt vet clean

help: ## show available targets
	@awk 'BEGIN { FS = ":.*##" } /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## compile binary into $(BIN_DIR) (default ~/.local/bin)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) $(PKG)
	@echo "orbit installed: $(BIN)"
	@case ":$$PATH:" in *":$(BIN_DIR):"*) ;; *) echo "note: $(BIN_DIR) is not in PATH" ;; esac

install: build ## alias for build (binary lives outside the repo)

test: ## run unit tests
	$(GO) test ./...

smoke: build ## run end-to-end smoke test
	ORBIT=$(BIN) ./scripts/smoke.sh

tidy: ## tidy go.mod / go.sum
	$(GO) mod tidy

fmt: ## gofmt
	$(GO) fmt ./...

vet: ## go vet
	$(GO) vet ./...

clean: ## remove installed binary
	rm -f $(BIN)
	@echo "removed: $(BIN)"
