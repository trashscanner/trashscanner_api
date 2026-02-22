run:
	go run cmd/trashscanner/main.go
	
mock-gen:
	if [ -z $$(which mockery 2>/dev/null) ]; then \
    	go install github.com/vektra/mockery/v3@v3.5.5; \
	fi
	mockery

lint:
	if [ -z $$(which golangci-lint 2>/dev/null) ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.5.0; \
	fi
	golangci-lint cache clean
	golangci-lint run --timeout 5m --config golang-ci.yaml

test:
	go test $$(go list ./... | grep -v '/internal/database' | grep -v '/docs' | grep -v '/models' | \
		grep -v '/internal/api/dto'  | grep -v '/mocks' | grep -v '/filestore' | grep -v '/cmd/' | grep -v '/tests/') \
		-coverprofile=coverage.out --race --timeout 2m
	cat coverage.out | grep -v "internal/database/sqlc" > coverage.txt || true

test-all:
	go test $$(go list ./... | grep -v '/mocks' | grep -v '/tests/') --race --timeout 2m

test-db:
	go test ./internal/database/... --race --timeout 2m

test-filestore:
	go test ./internal/filestore/... --race --timeout 2m

test-integration:
	go test ./tests/integration/... -v --race --timeout 5m

build:
	go build -o bin/trashscanner cmd/trashscanner/main.go

sqlc-gen:
	@if [ -z $$(which sqlc 2>/dev/null) ]; then \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest; \
	fi
	cd internal/database/sqlc && sqlc generate
	cd ../../..

new-migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required. Usage: make new-migration name=your_migration_name"; \
		exit 1; \
	fi; \
	LAST_NUM=$$(ls internal/database/migrations/ 2>/dev/null | grep -E '^[0-9]+_.*\.up\.sql$$' | sed 's/^\([0-9]*\)_.*/\1/' | sort -n | tail -1); \
	if [ -z "$$LAST_NUM" ]; then \
		LAST_NUM=0; \
	fi; \
	NEXT_NUM=$$(printf "%06d" $$(($$LAST_NUM + 1))); \
	UP_FILE="internal/database/migrations/$${NEXT_NUM}_$(name).up.sql"; \
	DOWN_FILE="internal/database/migrations/$${NEXT_NUM}_$(name).down.sql"; \
	touch $$UP_FILE $$DOWN_FILE; \
	echo "Created migration files:"; \
	echo "  - $$UP_FILE"; \
	echo "  - $$DOWN_FILE"

local-store:
	docker-compose -f docker-compose.local.yml up -d

connect-local-pg-store:
	docker exec -it trashscanner_postgres psql -U trashscanner -d trashscanner_db

drop-local-store:
	docker-compose -f docker-compose.local.yml down -v --remove-orphans

gen-keys:
	@ssh-keygen -t ed25519 -f /tmp/jwt_key -N "" -q
	@echo "Private Key:"
	@base64 -w 0 /tmp/jwt_key
	@echo ""
	@echo "Public Key:"
	@base64 -w 0 /tmp/jwt_key.pub
	@echo ""
	@rm /tmp/jwt_key /tmp/jwt_key.pub

swagger:
	if [ -z $$(which swag 2>/dev/null) ]; then \
		go install github.com/swaggo/swag/cmd/swag@v1.16.6; \
	fi
	swag init -g internal/api/server.go

.PHONY: run mock-gen lint test test-all test-db build sqlc-gen new-migration local-store connect-local-pg-store drop-local-store gen-keys swagger
