#!/usr/bin/env bash
# auth_functionality_challenge.sh - Validates Auth module core functionality
# Checks JWT, API key, OAuth types, middleware, and token management
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MODULE_NAME="Auth"

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

echo "=== ${MODULE_NAME} Functionality Challenge ==="
echo ""

# --- Section 1: Required packages ---
echo "Section 1: Required packages (5)"

for pkg in jwt apikey oauth middleware token; do
    echo "Test: Package pkg/${pkg} exists"
    if [ -d "${MODULE_DIR}/pkg/${pkg}" ]; then
        pass "Package pkg/${pkg} exists"
    else
        fail "Package pkg/${pkg} missing"
    fi
done

# --- Section 2: JWT ---
echo ""
echo "Section 2: JWT"

echo "Test: JWT Manager struct exists"
if grep -q "type Manager struct" "${MODULE_DIR}/pkg/jwt/"*.go 2>/dev/null; then
    pass "JWT Manager struct exists"
else
    fail "JWT Manager struct missing"
fi

echo "Test: JWT Token struct exists"
if grep -q "type Token struct" "${MODULE_DIR}/pkg/jwt/"*.go 2>/dev/null; then
    pass "JWT Token struct exists"
else
    fail "JWT Token struct missing"
fi

echo "Test: JWT Config struct exists"
if grep -q "type Config struct" "${MODULE_DIR}/pkg/jwt/"*.go 2>/dev/null; then
    pass "JWT Config struct exists"
else
    fail "JWT Config struct missing"
fi

echo "Test: JWT Parser interface exists"
if grep -q "type Parser interface" "${MODULE_DIR}/pkg/jwt/"*.go 2>/dev/null; then
    pass "JWT Parser interface exists"
else
    fail "JWT Parser interface missing"
fi

# --- Section 3: API Key ---
echo ""
echo "Section 3: API Key"

echo "Test: APIKey struct exists"
if grep -q "type APIKey struct" "${MODULE_DIR}/pkg/apikey/"*.go 2>/dev/null; then
    pass "APIKey struct exists"
else
    fail "APIKey struct missing"
fi

echo "Test: Generator struct exists"
if grep -q "type Generator struct" "${MODULE_DIR}/pkg/apikey/"*.go 2>/dev/null; then
    pass "Generator struct exists"
else
    fail "Generator struct missing"
fi

echo "Test: KeyStore interface exists"
if grep -q "type KeyStore interface" "${MODULE_DIR}/pkg/apikey/"*.go 2>/dev/null; then
    pass "KeyStore interface exists"
else
    fail "KeyStore interface missing"
fi

echo "Test: InMemoryStore struct exists"
if grep -q "type InMemoryStore struct" "${MODULE_DIR}/pkg/apikey/"*.go 2>/dev/null; then
    pass "InMemoryStore struct exists in apikey"
else
    fail "InMemoryStore struct missing in apikey"
fi

# --- Section 4: OAuth ---
echo ""
echo "Section 4: OAuth"

echo "Test: Credentials struct exists"
if grep -q "type Credentials struct" "${MODULE_DIR}/pkg/oauth/"*.go 2>/dev/null; then
    pass "OAuth Credentials struct exists"
else
    fail "OAuth Credentials struct missing"
fi

echo "Test: AutoRefresher struct exists"
if grep -q "type AutoRefresher struct" "${MODULE_DIR}/pkg/oauth/"*.go 2>/dev/null; then
    pass "AutoRefresher struct exists"
else
    fail "AutoRefresher struct missing"
fi

echo "Test: TokenRefresher interface exists"
if grep -q "type TokenRefresher interface" "${MODULE_DIR}/pkg/oauth/"*.go 2>/dev/null; then
    pass "TokenRefresher interface exists"
else
    fail "TokenRefresher interface missing"
fi

# --- Section 5: Middleware ---
echo ""
echo "Section 5: Middleware"

echo "Test: TokenValidator interface exists"
if grep -q "type TokenValidator interface" "${MODULE_DIR}/pkg/middleware/"*.go 2>/dev/null; then
    pass "TokenValidator interface exists"
else
    fail "TokenValidator interface missing"
fi

echo "Test: APIKeyValidator interface exists"
if grep -q "type APIKeyValidator interface" "${MODULE_DIR}/pkg/middleware/"*.go 2>/dev/null; then
    pass "APIKeyValidator interface exists"
else
    fail "APIKeyValidator interface missing"
fi

# --- Section 6: Token management ---
echo ""
echo "Section 6: Token management"

echo "Test: Token interface exists"
if grep -q "type Token interface" "${MODULE_DIR}/pkg/token/"*.go 2>/dev/null; then
    pass "Token interface exists"
else
    fail "Token interface missing"
fi

echo "Test: Store interface exists"
if grep -q "type Store interface" "${MODULE_DIR}/pkg/token/"*.go 2>/dev/null; then
    pass "Store interface exists"
else
    fail "Store interface missing"
fi

# --- Section 7: Source structure completeness ---
echo ""
echo "Section 7: Source structure"

echo "Test: Each package has non-test Go source files"
all_have_source=true
for pkg in jwt apikey oauth middleware token; do
    non_test=$(find "${MODULE_DIR}/pkg/${pkg}" -name "*.go" ! -name "*_test.go" -type f 2>/dev/null | wc -l)
    if [ "$non_test" -eq 0 ]; then
        fail "Package pkg/${pkg} has no non-test Go files"
        all_have_source=false
    fi
done
if [ "$all_have_source" = true ]; then
    pass "All packages have non-test Go source files"
fi

echo ""
echo "=== Results: ${PASS}/${TOTAL} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ] && exit 0 || exit 1
