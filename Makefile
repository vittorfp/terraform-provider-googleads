SHELL := /usr/bin/env bash
NAMESPACE := vittorfp
NAME := googleads
BINARY := terraform-provider-$(NAME)
VERSION := 0.6.0
HOSTNAME := registry.terraform.io
OS_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR := $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

.PHONY: build install fmt lint test testacc docs tidy clean

build:
	go build -o $(BINARY)

install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY) $(INSTALL_DIR)/

fmt:
	go fmt ./...
	@command -v terraform >/dev/null && terraform fmt -recursive examples/ || true

lint:
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "golangci-lint not installed; skipping"
	go vet ./...

test:
	go test -v -race ./...

testacc:
	TF_ACC=1 go test -v -race -timeout 60m ./internal/provider

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.20.1 generate \
		--provider-name $(NAME) --rendered-provider-name $(NAME)

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf dist/
