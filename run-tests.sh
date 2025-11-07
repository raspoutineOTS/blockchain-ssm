#!/bin/bash
#
# Complete Test Suite Runner
# Runs all tests for blockchain-ssm
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

print_header() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                                                        ║${NC}"
    echo -e "${BLUE}║     Blockchain-SSM Complete Test Suite                ║${NC}"
    echo -e "${BLUE}║                                                        ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_section() {
    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}  $1${NC}"
}

run_test() {
    local test_name="$1"
    local test_command="$2"

    echo ""
    print_info "Running: $test_name"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if eval "$test_command"; then
        print_success "$test_name passed"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_error "$test_name failed"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

print_summary() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                    TEST SUMMARY                        ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "  Total Tests: $TOTAL_TESTS"
    echo -e "  ${GREEN}Passed: $PASSED_TESTS${NC}"

    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "  ${RED}Failed: $FAILED_TESTS${NC}"
    else
        echo -e "  ${GREEN}Failed: 0${NC}"
    fi

    echo ""

    local pass_rate=0
    if [ $TOTAL_TESTS -gt 0 ]; then
        pass_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    fi

    echo "  Pass Rate: ${pass_rate}%"
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║           ✓ ALL TESTS PASSED ✓                        ║${NC}"
        echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
        return 0
    else
        echo -e "${RED}╔════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║           ✗ SOME TESTS FAILED ✗                       ║${NC}"
        echo -e "${RED}╚════════════════════════════════════════════════════════╝${NC}"
        return 1
    fi
}

# Main execution
print_header

# Check prerequisites
print_section "Checking Prerequisites"

if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    exit 1
fi
print_success "Go installed: $(go version)"

if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed"
    exit 1
fi
print_success "Node.js installed: $(node --version)"

if ! command -v npm &> /dev/null; then
    print_error "npm is not installed"
    exit 1
fi
print_success "npm installed: $(npm --version)"

# Install dependencies
print_section "Installing Dependencies"

print_info "Installing Go dependencies..."
cd chaincode/go/ssm
if go mod download; then
    print_success "Go dependencies installed"
else
    print_error "Failed to install Go dependencies"
    exit 1
fi
cd - > /dev/null

print_info "Installing npm dependencies..."
cd test
if [ ! -d "node_modules" ]; then
    if npm install; then
        print_success "npm dependencies installed"
    else
        print_error "Failed to install npm dependencies"
        exit 1
    fi
else
    print_success "npm dependencies already installed"
fi
cd - > /dev/null

# Run Go tests
print_section "Go Unit Tests"

cd chaincode/go/ssm
run_test "Go Build" "go build"
run_test "Go Unit Tests" "go test -v -race -short ./... 2>&1 | tee /tmp/go-test.log || true"

# Check if tests passed
if grep -q "FAIL" /tmp/go-test.log; then
    print_error "Some Go tests failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
elif grep -q "PASS" /tmp/go-test.log; then
    print_success "All Go tests passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

cd - > /dev/null

# Run JavaScript integration tests
print_section "JavaScript Integration Tests"

cd test

run_test "ITMO Integration Tests" "npm run test:itmo"
run_test "DeFi Integration Tests" "npm run test:defi"

cd - > /dev/null

# Code quality checks
print_section "Code Quality Checks"

print_info "Checking Go formatting..."
cd chaincode/go/ssm
UNFORMATTED=$(gofmt -l .)
if [ -z "$UNFORMATTED" ]; then
    print_success "Go code is properly formatted"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "Some Go files are not formatted:"
    echo "$UNFORMATTED"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd - > /dev/null

# Documentation checks
print_section "Documentation Checks"

REQUIRED_DOCS=(
    "README.md"
    "ITMO_README.md"
    "DEFI_README.md"
    "test/README.md"
    "CARBON_REGISTRY_INTEGRATION.md"
    "ITMO_ARTICLE6_ARCHITECTURE.md"
)

TOTAL_TESTS=$((TOTAL_TESTS + 1))
DOC_MISSING=0

for doc in "${REQUIRED_DOCS[@]}"; do
    if [ -f "$doc" ]; then
        print_success "Found: $doc"
    else
        print_error "Missing: $doc"
        DOC_MISSING=1
    fi
done

if [ $DOC_MISSING -eq 0 ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
    print_success "All required documentation present"
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    print_error "Some documentation missing"
fi

# Print summary
print_summary
exit $?
