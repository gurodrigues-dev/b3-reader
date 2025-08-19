.PHONY: test
test:
	@go test -short -coverprofile=cp.out $$(go list ./... | grep -vE cmd\|config)

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: update
update:
	@go get -u ./...
	@go mod tidy

.PHONY: vulncheck
vulncheck:
	@govulncheck ./...

.PHONY: vet
vet:
	@go vet ./...

# build de aplicacao

# build de ingestor

# build de docker-compose


