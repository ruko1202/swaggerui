# GOBIN - Directory for installed Go binaries
# Default: ./bin in the project root
GOBIN			?= $(PWD)/bin
# Ensure GOBIN directory exists
$(shell mkdir -p $(GOBIN))

.PHONY: update-assets
update-assets:
	go run ./generator/generate.go

.PHONY: bin-deps
bin-deps:
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.2

.PHONY: lint
lint:
	$(GOBIN)/golangci-lint run

.PHONY: tloc
tloc:
	go test -p 2 -count 2 --race --coverpkg=./... --covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sed 's|github.com/ruko1202/swaggerui||' | sed -E 's/\t+/\t/g' | tee coverage.report
