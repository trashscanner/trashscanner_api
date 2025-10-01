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
	golangci-lint run

test:
	go test ./... -coverprofile=coverage.txt --race --timeout 2m

build:
	go build -o bin/trashscanner cmd/trashscanner/main.go

.PHONY: run mock-gen lint test
