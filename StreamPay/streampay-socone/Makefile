# Project Configuration
APP_NAME ?= streampay
DAEMON_NAME = $(APP_NAME)d
CHAIN_ID = scop-chain
BUILDDIR ?= $(CURDIR)/build
LEDGER_ENABLED ?= true
GO_SYSTEM_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1-2)
REQUIRE_GO_VERSION = 1.20
DOCKER := $(shell which docker)

# Git Information
GIT_URL ?= $(shell git config --get remote.origin.url)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TAG ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo "dev")


# Build Tags and Linker Flags
build_tags = netgo
build_tags += ledger
build_tags += cleveldb
build_tags := $(strip $(build_tags))

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=$(APP_NAME) \
          -X github.com/cosmos/cosmos-sdk/version.AppName=$(DAEMON_NAME) \
          -X github.com/cosmos/cosmos-sdk/version.Version=$(TAG) \
          -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)' -trimpath

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

# Commands
.PHONY: all install build test lint clean reset start \
        image push help deps sync

all: install ## Run all tasks

install: go.sum ## Install the binary
	@go install -mod=readonly $(BUILD_FLAGS) ./cmd/$(DAEMON_NAME)

build: ## Build the binary
	@go build $(BUILD_FLAGS) -o $(BUILDDIR)/$(DAEMON_NAME) ./cmd/$(DAEMON_NAME)

test: ## Run tests and generate coverage report
	@go test -v -cover -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

lint: ## Run linter
	@golangci-lint run

clean: ## Clean build artifacts and coverage files
	@rm -rf $(BUILDDIR) coverage.out

deps: ## Install project dependencies
	@go mod tidy
	@go mod download

proto-gen:
	@echo "Generating protobuf files..."
	@$(protoImage) sh ./scripts/protocgen.sh
	@go mod tidy

.PHONY: release
release: ## build binaries and docker image for different platform 
	@goreleaser release --config goreleaser.yml --clean --snapshot

.PHONY: image
# Build Docker image with metadata labels
image: ## build docker image
	@docker build \
		--build-arg GIT_URL=$(GIT_URL) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_BRANCH=$(GIT_BRANCH) \
		--build-arg DATE=$(DATE) \
		--build-arg TAG=$(TAG) \
		-t minhbui1/$(APP_NAME):$(TAG) \
		-t minhbui1/$(APP_NAME):latest .

push: ## Push Docker image to registry
#   @docker push minhbui1/$(APP_NAME):$(TAG)
	@docker push minhbui1/$(APP_NAME):latest

# reset1: ## Reset and start a test chain
# 	@rm -rf ~/.$(APP_NAME)/config/*
# 	@$(DAEMON_NAME) tendermint unsafe-reset-all
# 	@$(DAEMON_NAME) init scop-node --chain-id "$(CHAIN_ID)"
# 	@$(DAEMON_NAME) keys add validator --keyring-backend test
# 	@$(DAEMON_NAME) keys add user1 --keyring-backend test
# 	@$(DAEMON_NAME) keys add user2 --keyring-backend test
# 	@$(DAEMON_NAME) genesis add-genesis-account `$(DAEMON_NAME) keys show validator -a --keyring-backend test` 1000000000stake,1000000000000sco
# 	@$(DAEMON_NAME) genesis add-genesis-account `$(DAEMON_NAME) keys show user1 -a --keyring-backend test` 1000000000stake,1000000000000sco
# 	@$(DAEMON_NAME) genesis add-genesis-account `$(DAEMON_NAME) keys show user2 -a --keyring-backend test` 1000000000stake,1000000000000sco
# 	@$(DAEMON_NAME) genesis gentx validator 1000000000stake --moniker "validator-1" --chain-id "$(CHAIN_ID)" --keyring-backend test
# 	@$(DAEMON_NAME) genesis collect-gentxs
# 	@$(DAEMON_NAME) genesis validate-genesis
# 	@$(DAEMON_NAME) start --minimum-gas-prices 0.0stake --api.enable --grpc-web.enable --grpc.enable --api.enabled-unsafe-cors --rpc.unsafe --api.swagger

# start1: ## Reset and start a test chain
# 	@$(DAEMON_NAME) start --minimum-gas-prices 0.0stake --api.enable --grpc-web.enable --grpc.enable --api.enabled-unsafe-cors --rpc.unsafe --api.swagger

reset:
	@./int_network.sh

start: ## Reset and start a test chain
	@docker-compose up -d


help: ## Display this help message
	@echo "Usage: make [target]"
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

go.sum: go.mod ## Ensure dependencies have not been modified
	@echo "--> Verifying dependencies"
	GO111MODULE=on go mod verify

sync: ## sync latest source code from github
	@./scripts/sync.sh
