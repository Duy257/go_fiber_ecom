#!/usr/bin/env bash
# ===========================================
# API Test Script — Go Fiber
# Usage: bash docs/curl.sh
# ===========================================
set -euo pipefail

BASE="${BASE:-http://localhost:3000/api/v1}"

# ── Colors ─────────────────────────────────
R='\033[0;31m'
G='\033[0;32m'
Y='\033[1;33m'
B='\033[0;34m'
C='\033[0;36m'
NC='\033[0m' # No Color

ok()   { echo -e "  ${G}✓${NC} $1"; }
info() { echo -e "  ${C}→${NC} $1"; }
warn() { echo -e "  ${Y}⚠${NC} $1"; }
fail() { echo -e "  ${R}✗${NC} $1"; }
sep()  { echo -e "${B}──────────────────────────────────────────────${NC}"; }
title(){ echo; echo -e "${B}══ $1 ══${NC}"; }

# ── Helpers ────────────────────────────────
TOKEN_FILE=$(mktemp)
trap 'rm -f $TOKEN_FILE' EXIT

save_tokens() {
  echo "$1" > "$TOKEN_FILE"
}

get_access() {
  if [[ -f "$TOKEN_FILE" ]]; then
    cat "$TOKEN_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('access_token',''))" 2>/dev/null || echo ""
  fi
}

run_curl() {
  local method="$1" path="$2" desc="$3" bearer="${4:-}" body="${5:-}"
  local auth_flag=""
  local full_url="${BASE}${path}"

  if [[ -n "$bearer" ]]; then
    auth_flag="-H Authorization: Bearer ${bearer}"
  fi

  echo -e "  ${Y}${method}${NC} ${full_url}"
  if [[ -n "$desc" ]]; then
    echo -e "  ${C}# ${desc}${NC}"
  fi

  local curl_cmd="curl -s -X ${method} '${full_url}' ${auth_flag} -H 'Content-Type: application/json'"
  if [[ -n "$body" ]]; then
    curl_cmd="${curl_cmd} -d '${body}'"
  fi

  echo "  ${curl_cmd}"
  echo

  # Execute and pretty-print
  if [[ -n "$body" ]]; then
    curl -s -X "$method" "${full_url}" ${auth_flag} -H 'Content-Type: application/json' -d "${body}" | python3 -m json.tool 2>/dev/null || curl -s -X "$method" "${full_url}" ${auth_flag} -H 'Content-Type: application/json' -d "${body}"
  else
    curl -s -X "$method" "${full_url}" ${auth_flag} -H 'Content-Type: application/json' | python3 -m json.tool 2>/dev/null || curl -s -X "$method" "${full_url}" ${auth_flag} -H 'Content-Type: application/json'
  fi

  echo
}


# ════════════════════════════════════════════
title "AUTH — Admin Login"
sep
RESP=$(curl -s -X POST "${BASE}/auth/admin/login" \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin@gmail.com","password":"123123"}')
echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"
save_tokens "$RESP"
ACCESS=$(get_access)
ok "Access token saved"

# ────────────────────────────────────────────
title "AUTH — Customer Login"
sep
CUST_RESP=$(curl -s -X POST "${BASE}/auth/customer/login" \
  -H 'Content-Type: application/json' \
  -d '{"login":"customer@example.com","password":"123123"}')
echo "$CUST_RESP" | python3 -m json.tool 2>/dev/null || echo "$CUST_RESP"
CUST_ACCESS=$(echo "$CUST_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('access_token',''))" 2>/dev/null || echo "")
ok "Customer token saved"

# ────────────────────────────────────────────
title "AUTH — Refresh Token"
sep
REFRESH=$(cat "$TOKEN_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('refresh_token',''))" 2>/dev/null || echo "")
if [[ -n "$REFRESH" ]]; then
  curl -s -X POST "${BASE}/auth/refresh" \
    -H 'Content-Type: application/json' \
    -d "{\"refresh_token\":\"${REFRESH}\"}" | python3 -m json.tool 2>/dev/null || echo "$RESP"
fi

# ────────────────────────────────────────────
title "CUSTOMER — Profile (self-service)"
sep
run_curl "GET" "/customer/profile" "Get own profile" "$CUST_ACCESS"

# ────────────────────────────────────────────
title "CUSTOMER — Update Profile"
sep
run_curl "PUT" "/customer/profile" "Update profile" "$CUST_ACCESS" \
  '{"name":"Updated Customer","phone_number":"0335909201"}'

# ────────────────────────────────────────────
title "DASHBOARD — Stats"
sep
run_curl "GET" "/admin/dashboard/stats" "Requires dashboard:read" "$ACCESS"

# ════════════════════════════════════════════
title "CUSTOMERS CRUD"
sep

# ── List ───────────────────────────────────
run_curl "GET" "/admin/customers?page=1&limit=10" "List customers (customer:read)" "$ACCESS"

# ── Create ─────────────────────────────────
run_curl "POST" "/admin/customers" "Create customer (customer:write)" "$ACCESS" \
  '{"name":"Test Customer","email":"testcurl@example.com","phone_number":"0999999999","password":"123456"}'

# ── Get by ID (use the ID from create) ─────
CUST_ID=$(curl -s -X GET "${BASE}/admin/customers?page=1&limit=1" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',[]); print(d[0]['id'] if d else '')" 2>/dev/null || echo "")

if [[ -n "$CUST_ID" ]]; then
  run_curl "GET" "/admin/customers/${CUST_ID}" "Get customer by ID" "$ACCESS"

  # ── Update ───────────────────────────────
  run_curl "PUT" "/admin/customers/${CUST_ID}" "Update customer" "$ACCESS" \
    '{"name":"Updated Via Curl","status":"active"}'

  # ── Delete ───────────────────────────────
  run_curl "DELETE" "/admin/customers/${CUST_ID}" "Delete customer (customer:delete)" "$ACCESS"
else
  warn "No customer found to test get/update/delete"
fi

# ════════════════════════════════════════════
title "USERS CRUD"
sep

run_curl "GET" "/admin/users?page=1&limit=10" "List users (user:read)" "$ACCESS"

# ── Get first role_id for user creation ────
ROLE_ID=$(curl -s -X GET "${BASE}/admin/roles" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',[]); print(d[0]['id'] if d else '')" 2>/dev/null || echo "")

if [[ -n "$ROLE_ID" ]]; then
  run_curl "POST" "/admin/users" "Create user (user:write)" "$ACCESS" \
    "{\"name\":\"Staff Curl\",\"email\":\"staffcurl@example.com\",\"password\":\"123456\",\"role_id\":\"${ROLE_ID}\"}"
else
  warn "No role found, skipping user create"
fi

USER_ID=$(curl -s -X GET "${BASE}/admin/users?page=1&limit=1" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',[]); print(d[0]['id'] if d else '')" 2>/dev/null || echo "")

if [[ -n "$USER_ID" ]]; then
  run_curl "GET" "/admin/users/${USER_ID}" "Get user by ID" "$ACCESS"
  run_curl "PUT" "/admin/users/${USER_ID}" "Update user" "$ACCESS" \
    '{"name":"Updated Staff","status":"active"}'
  # run_curl "DELETE" "/admin/users/${USER_ID}" "Delete user" "$ACCESS"
else
  warn "No user found"
fi

# ════════════════════════════════════════════
title "ROLES CRUD"
sep

run_curl "GET" "/admin/roles" "List roles (role:read)" "$ACCESS"

# ── Get permission IDs ─────────────────────
PERM_IDS=$(curl -s -X GET "${BASE}/admin/permissions" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  | python3 -c "
import sys,json
d=json.load(sys.stdin).get('data',[])
ids=[p['id'] for p in d[:3]]
print(json.dumps(ids))
" 2>/dev/null || echo "[]")

run_curl "POST" "/admin/roles" "Create role (role:write)" "$ACCESS" \
  "{\"name\":\"curl_role\",\"description\":\"Role created via curl\",\"permission_ids\":${PERM_IDS}}"

ROLE_ID2=$(curl -s -X GET "${BASE}/admin/roles" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',[]); print(d[-1]['id'] if d else '')" 2>/dev/null || echo "")

if [[ -n "$ROLE_ID2" ]]; then
  run_curl "PUT" "/admin/roles/${ROLE_ID2}" "Update role" "$ACCESS" \
    '{"name":"curl_role_updated","description":"Updated description"}'
  run_curl "DELETE" "/admin/roles/${ROLE_ID2}" "Delete role (role:delete)" "$ACCESS"
fi

# ════════════════════════════════════════════
title "PERMISSIONS"
sep

run_curl "GET" "/admin/permissions" "List permissions (permission:read)" "$ACCESS"
run_curl "POST" "/admin/permissions" "Create permission (permission:write)" "$ACCESS" \
  '{"name":"report:read","description":"View reports"}'

# ════════════════════════════════════════════
title "ERROR CASES"
sep

echo -e "  ${Y}POST${NC} ${BASE}/auth/admin/login (wrong password)"
curl -s -X POST "${BASE}/auth/admin/login" \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin@gmail.com","password":"wrong"}' | python3 -m json.tool 2>/dev/null
echo

echo -e "  ${Y}GET${NC} ${BASE}/admin/customers (no token)"
curl -s -X GET "${BASE}/admin/customers" \
  -H 'Content-Type: application/json' | python3 -m json.tool 2>/dev/null
echo

echo -e "  ${Y}GET${NC} ${BASE}/admin/customers (wrong token)"
curl -s -X GET "${BASE}/admin/customers" \
  -H 'Authorization: Bearer invalidtoken' \
  -H 'Content-Type: application/json' | python3 -m json.tool 2>/dev/null
echo

echo -e "  ${Y}POST${NC} ${BASE}/admin/customers (no body)"
curl -s -X POST "${BASE}/admin/customers" \
  -H "Authorization: Bearer ${ACCESS}" \
  -H 'Content-Type: application/json' \
  -d '{}' | python3 -m json.tool 2>/dev/null
echo

# ── Cleanup ────────────────────────────────
echo
sep
echo -e "${G}Done.${NC}"
