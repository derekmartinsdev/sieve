.PHONY: build
build:
	go build -o transpiler ./cmd/transpiler/

.PHONY: test
test:
	go test ./... -count=1

.PHONY: test-verbose
test-verbose:
	go test ./... -v -count=1

.PHONY: test-cover
test-cover:
	go test ./... -cover -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

.PHONY: test-cover-html
test-cover-html: test-cover
	go tool cover -html=coverage.out -o coverage.html

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run --timeout 60s

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix --timeout 60s

.PHONY: sec
sec:
	gosec -quiet ./...

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: ci
ci: tidy fmt vet lint sec test-cover build

.PHONY: quality-gate
quality-gate:
	@echo "=========================================="
	@echo "  QUALITY GATE — Spark DSL Transpiler"
	@echo "=========================================="
	@echo ""
	@echo "[1/6] gofmt..."
	@test -z "$$(gofmt -l .)" || (echo "❌  gofmt: unformatted files found" && gofmt -l . && exit 1)
	@echo "✅  gofmt: all files formatted"
	@echo ""
	@echo "[2/6] go vet..."
	@go vet ./... || (echo "❌  go vet: issues found" && exit 1)
	@echo "✅  go vet: clean"
	@echo ""
	@echo "[3/6] golangci-lint..."
	@golangci-lint run --timeout 60s || (echo "❌  lint: issues found" && exit 1)
	@echo "✅  lint: 0 issues"
	@echo ""
	@echo "[4/6] gosec..."
	@gosec -quiet ./... > /dev/null 2>&1 || (echo "❌  gosec: security issues found" && gosec ./... && exit 1)
	@echo "✅  gosec: no security issues"
	@echo ""
	@echo "[5/6] go test + coverage..."
	@go test ./... -cover -coverprofile=coverage.out -count=1 > /tmp/test-output.txt 2>&1
	@if grep -q "^--- FAIL" /tmp/test-output.txt; then \
		echo "❌  tests: failures found" && cat /tmp/test-output.txt && exit 1; \
	fi
	@echo "✅  tests: all passing"
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -func=coverage.out | awk 'END {total=$$NF; gsub("%","",total); if(total+0<60) {print "❌  Coverage below 60%"; exit 1} else {print "✅  Coverage ≥60%"; exit 0}}'
	@echo ""
	@echo "[6/6] go build..."
	@go build -o transpiler ./cmd/transpiler/ || (echo "❌  build: compilation failed" && exit 1)
	@echo "✅  build: successful"
	@echo ""
	@echo "=========================================="
	@echo "  QUALITY GATE: PASSED ✅"
	@echo "=========================================="

.PHONY: clean
clean:
	rm -f transpiler coverage.out coverage.html