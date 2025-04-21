# Project Configuration
APP_NAME ?= dexplorer
DOCKER := $(shell which docker)

# Git Information
GIT_URL ?= $(shell git config --get remote.origin.url)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TAG ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo "dev")

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

.PHONY: push
push: ## Push Docker image to registry
	@docker push minhbui1/$(APP_NAME):$(TAG)
	@docker push minhbui1/$(APP_NAME):latest

help: ## Display this help message
	@echo "Usage: make [target]"
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
