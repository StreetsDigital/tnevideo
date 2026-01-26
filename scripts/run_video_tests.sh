#!/bin/bash

# Video Test Suite Execution Script
# This script runs the complete video functionality test suite and generates coverage reports

set -e

echo "========================================="
echo "TNEVideo - Video Test Suite"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test results directory
RESULTS_DIR="test-results"
mkdir -p $RESULTS_DIR

echo -e "${YELLOW}[1/7] Running VAST unit tests...${NC}"
go test ./pkg/vast/... -v -cover -coverprofile=$RESULTS_DIR/vast_coverage.out || {
    echo -e "${RED}VAST unit tests failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ VAST unit tests passed${NC}"
echo ""

echo -e "${YELLOW}[2/7] Running video handler unit tests...${NC}"
go test ./internal/endpoints/video_* -v -cover -coverprofile=$RESULTS_DIR/handler_coverage.out || {
    echo -e "${RED}Handler unit tests failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ Video handler tests passed${NC}"
echo ""

echo -e "${YELLOW}[3/7] Running video integration tests...${NC}"
go test -tags=integration ./tests/integration/video_* -v -timeout 30s || {
    echo -e "${RED}Integration tests failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ Integration tests passed${NC}"
echo ""

echo -e "${YELLOW}[4/7] Running OpenRTB compliance tests...${NC}"
go test -tags=integration ./tests/integration/openrtb_video_compliance_test.go -v || {
    echo -e "${RED}OpenRTB compliance tests failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ OpenRTB compliance tests passed${NC}"
echo ""

echo -e "${YELLOW}[5/7] Running video benchmarks...${NC}"
go test -bench=. -benchmem ./tests/benchmark/video_benchmark_test.go -run=^$ | tee $RESULTS_DIR/benchmark_results.txt
echo -e "${GREEN}✓ Benchmarks completed${NC}"
echo ""

echo -e "${YELLOW}[6/7] Checking for race conditions...${NC}"
go test -race ./pkg/vast/... || {
    echo -e "${RED}Race conditions detected in VAST package!${NC}"
    exit 1
}
go test -tags=integration -race -timeout 60s ./tests/integration/video_outbound_test.go ./tests/integration/video_inbound_test.go || {
    echo -e "${RED}Race conditions detected in integration tests!${NC}"
    exit 1
}
echo -e "${GREEN}✓ No race conditions detected${NC}"
echo ""

echo -e "${YELLOW}[7/7] Generating coverage reports...${NC}"

# Combine coverage files
echo "mode: set" > $RESULTS_DIR/combined_coverage.out
tail -q -n +2 $RESULTS_DIR/*_coverage.out >> $RESULTS_DIR/combined_coverage.out 2>/dev/null || true

# Generate HTML coverage report
go tool cover -html=$RESULTS_DIR/combined_coverage.out -o $RESULTS_DIR/coverage.html

# Calculate total coverage
COVERAGE=$(go tool cover -func=$RESULTS_DIR/combined_coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}Total coverage: $COVERAGE${NC}"

# Check if coverage meets minimum threshold (80%)
COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
THRESHOLD=80

if (( $(echo "$COVERAGE_NUM >= $THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage meets ${THRESHOLD}% threshold${NC}"
else
    echo -e "${RED}✗ Coverage ${COVERAGE} is below ${THRESHOLD}% threshold${NC}"
    exit 1
fi

echo ""
echo "========================================="
echo -e "${GREEN}All tests passed!${NC}"
echo "========================================="
echo ""
echo "Coverage report: $RESULTS_DIR/coverage.html"
echo "Benchmark results: $RESULTS_DIR/benchmark_results.txt"
echo ""
echo "To view coverage report:"
echo "  open $RESULTS_DIR/coverage.html"
echo ""
