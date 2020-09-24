
GOLANGCI_LINT_VERSION=v1.27.0

LINTER=./bin/golangci-lint
LINTER_VERSION_FILE=./bin/.golangci-lint-version-$(GOLANGCI_LINT_VERSION)

ALL_SOURCES := $(shell find * -type f -name "*.go")

COVERAGE_PROFILE_RAW=./build/coverage_raw.out
COVERAGE_PROFILE_RAW_HTML=./build/coverage_raw.html
COVERAGE_PROFILE_FILTERED=./build/coverage.out
COVERAGE_PROFILE_FILTERED_HTML=./build/coverage.html
COVERAGE_ENFORCER_FLAGS=-package gopkg.in/launchdarkly/go-server-sdk-evaluation.v2 -skipcode "// COVERAGE" -packagestats -filestats -showcode

TEST_BINARY=./go-server-sdk-evaluation.test
ALLOCATIONS_LOG=./allocations.out

.PHONY: build clean test lint test-coverage benchmarks benchmark-allocs

build:
	go build ./...

clean:
	go clean

test:
	go test -race -v ./...

test-coverage: $(COVERAGE_PROFILE_RAW)
	if [ -z "$(which go-coverage-enforcer)" ]; then go get github.com/launchdarkly-labs/go-coverage-enforcer; fi
	go-coverage-enforcer $(COVERAGE_ENFORCER_FLAGS) -outprofile $(COVERAGE_PROFILE_FILTERED) $(COVERAGE_PROFILE_RAW)
	go tool cover -html $(COVERAGE_PROFILE_FILTERED) -o $(COVERAGE_PROFILE_FILTERED_HTML)
	go tool cover -html $(COVERAGE_PROFILE_RAW) -o $(COVERAGE_PROFILE_RAW_HTML)

$(COVERAGE_PROFILE_RAW): $(ALL_SOURCES)
	@mkdir -p ./build
	go test -coverprofile $(COVERAGE_PROFILE_RAW) ./... >/dev/null

benchmarks:
	go test -benchmem '-run=^$$' '-bench=.*' ./...

# See CONTRIBUTING.md regarding the use of the benchmark-allocs target. Notes about this implementation:
# 1. We precompile the test code because otherwise the allocation traces will include the actions of the compiler itself.
# 2. "benchtime=3x" means the run count (b.N) is set to 3. Setting it to 1 would produce less redundant output, but the
#    benchmark statistics seem to be less reliable if the run count is less than 3 - they will report a higher allocation
#    count per run, possibly because the first run
benchmark-allocs:
	@if [ -z "$$BENCHMARK" ]; then echo "must specify BENCHMARK=" && exit 1; fi
	@echo Precompiling test code to $(TEST_BINARY)
	@go test -c -o $(TEST_BINARY) >/dev/null 2>&1
	@echo "Generating heap allocation traces in $(ALLOCATIONS_LOG) for benchmark(s): $$BENCHMARK"
	@echo "You should see some benchmark result output; if you do not, you may have misspelled the benchmark name/regex"
	@GODEBUG=allocfreetrace=1 $(TEST_BINARY) -test.run=none -test.bench=$$BENCHMARK -test.benchmem -test.benchtime=1x 2>$(ALLOCATIONS_LOG)

$(LINTER_VERSION_FILE):
	rm -f $(LINTER)
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s $(GOLANGCI_LINT_VERSION)
	touch $(LINTER_VERSION_FILE)

lint: $(LINTER_VERSION_FILE)
	$(LINTER) run ./...
