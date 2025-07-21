build-MainFunction: check-format test build

check-format:
	@echo "--- Checking format... ---"
	test -z $(shell gofmt -l .) || (echo "ERROR: Go files are not formatted. Please run 'go fmt ./...'"; exit 1)

test:
	@echo "--- Running unit tests... ---"
	go test -v

build:
	@echo "--- Building executable... ---"
	GOOS=linux GOARCH=arm64 go build -o $(ARTIFACTS_DIR)/bootstrap .

.PHONY: build test check-format
