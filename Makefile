.PHONY: build dependencies test test-race test-cover lint lint-ci tidy man clean install help

help:
	@echo "🐷 Available:"
	@echo "  build       - Build the slop binary"
	@echo "  dependencies- Download all dependencies"
	@echo "  test        - Run all tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  test-cover  - Run tests with coverage report"
	@echo "  lint        - Run linters (go vet, gofmt)"
	@echo "  lint-ci     - Run golangci-lint (CI-grade linting)"
	@echo "  tidy        - Clean up dependencies"
	@echo "  man         - Generate man pages"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install slop to GOPATH/bin"

build:
	$(info 🐷 building slop )
	go build -ldflags="-s -w" -o slop ./cmd/slop

dependencies:
	$(info 🐷 download dependencies  )
	go mod download
	go mod verify

test:
	$(info 🐷 running tests )
	go test -v ./...

test-race:
	$(info 🐷 running tests with race detection  )
	go test -v -race ./...

test-cover:
	$(info 🐷 running tests with coverage  )
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	$(info 🐷 checking code quality  )
	go vet ./...
	gofmt -l .
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi

	$(info 🐷 running golangci-lint  )
	golangci-lint run

tidy:
	$(info 🐷 clean up dependencies  )
	go mod tidy

man:
	$(info 🐷 building man pages  )
	go run ./cmd/slop man

clean:
	$(info 🐷 cleaning build artifacts  )
	rm -f slop
	rm -f coverage.out coverage.html
	rm -rf dist/

install: build
	$(info 🐷 installing slop 🐷 )
	go install ./cmd/slop