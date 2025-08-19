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

.PHONY: network
network:
	@if [ -z "$$($(COMPOSE) ls >/dev/null 2>&1; docker network ls --format '{{.Name}}' | grep -w $(NETWORK))" ]; then \
	  echo "Creating network $(NETWORK) ..."; \
	  docker network create $(NETWORK); \
	else \
	  echo "Network $(NETWORK) already exists."; \
	fi

.PHONY: build-api
build-api:
	@go build -o bin/api cmd/api/main.go

.PHONY: build-ingestor
build-ingestor:
	@go build -o bin/ingestor cmd/ingestor/main.go

.PHONY: ingestion
ingestion:
	@docker compose --profile ingestion up -d

.PHONY: ingestion-logs
ingestion-logs:
	@docker compose --profile ingestion up

.PHONY: api
api:
	@docker compose up -d

.PHONY: api-logs
api-logs:
	@docker compose up

