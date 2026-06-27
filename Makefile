GOLANGCI_LINT_VERSION := v2.2.2
GOLANGCI_LINT_MODULE := github.com/golangci/golangci-lint/v2/cmd/golangci-lint

.PHONY: install-lint lint test cover race

install-lint:
	go install $(GOLANGCI_LINT_MODULE)@$(GOLANGCI_LINT_VERSION)

lint:
	@golangci-lint --version
	@golangci-lint run

test:
	go test ./...

cover:
	go test -cover ./...

race:
	go test -race ./...
