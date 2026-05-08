.PHONY: build test test-cover snapshot bench bench-compare lint fmt clean run help

BIN      := sample_account
PKG      := ./cmd/sample_account
COVER    := coverage.out

help:
	@echo "Targets:"
	@echo "  build         - go build the binary"
	@echo "  test          - go test ./... -race"
	@echo "  test-cover    - go test ./... with coverage report"
	@echo "  snapshot      - run snapshot tests against tests/expected/"
	@echo "  bench         - go test -bench=. -benchmem ./..."
	@echo "  bench-compare - compare wall-clock vs C++ -O2 build"
	@echo "  lint          - golangci-lint run"
	@echo "  fmt           - gofumpt -w ."
	@echo "  clean         - remove build artifacts"

build:
	go build -trimpath -ldflags="-s -w" -o $(BIN) $(PKG)

test:
	go test ./... -race

test-cover:
	go test ./... -race -coverprofile=$(COVER) -covermode=atomic
	go tool cover -func=$(COVER) | tail -20

snapshot: build
	TZ=Asia/Tokyo go test -tags=snapshot ./tests/...

bench:
	go test -bench=. -benchmem -run=^$$ ./...

bench-compare: build
	@./scripts/bench-compare.sh

lint:
	golangci-lint run

fmt:
	gofumpt -w .

clean:
	rm -f $(BIN) $(COVER)
