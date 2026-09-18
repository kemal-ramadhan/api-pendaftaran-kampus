#!/usr/bin/env bash
# ==============================================================================
# PMB Backend - End-to-End Verification Script via API Gateway
# ==============================================================================
# Alur:
# 1. Register calon mahasiswa
# 2. Ambil kode OTP dari API Mailpit (http://localhost:8025/api/v1/messages)
# 3. Verifikasi OTP -> dapatkan token JWT (access & refresh)
# 4. Akses endpoint /me dengan access token
# 5. Refresh token -> peroleh access token & refresh token baru
# 6. Logout -> cabut sesi di Redis / denylist token
# 7. Akses kembali endpoint /me dengan token yang telah dicabut (harus 401)
# 8. Login sebagai superadmin
# 9. Superadmin membuat akun user dengan role keuangan
# 10. Login sebagai user keuangan
# 11. User keuangan mencoba mengakses endpoint admin (harus 403 Forbidden)
# ==============================================================================

set -e

# Konfigurasi Target URL
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8000/api/v1/auth}"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"

# Formatting & Warna Output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

pass_count=0
fail_count=0

log_info() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

log_pass() {
  echo -e "${GREEN}[PASS]${NC} $1"
  pass_count=$((pass_count + 1))
}

log_fail() {
  echo -e "${RED}[FAIL]${NC} $1"
  fail_count=$((fail_count + 1))
}

assert_status() {
  local step_name="$1"
  local expected="$2"
  local actual="$3"
  local body="$4"

  if [ "$actual" = "$expected" ]; then
    log_pass "$step_name (HTTP $actual)"
  else
    log_fail "$step_name - Expected HTTP $expected but got HTTP $actual"
    echo -e "${YELLOW}Response Body:${NC} $body"
    exit 1
  fi
}

echo "=============================================================================="
echo "          PMB BACKEND E2E TEST VIA API GATEWAY"
echo "=============================================================================="
echo "Gateway URL : $GATEWAY_URL"
echo "Mailpit URL : $MAILPIT_URL"
echo "Timestamp   : $(date)"
echo "------------------------------------------------------------------------------"

TIMESTAMP=$(date +%s)
RAND_ID="${RANDOM}_${RANDOM}"
MHS_EMAIL="calon_mhs_${RAND_ID}@example.com"
MHS_PASSWORD="Password123!"
MHS_NAME="Calon Mahasiswa E2E"
MHS_PHONE="081234567890"

# ------------------------------------------------------------------------------
# 1. Register Calon Mahasiswa
# ------------------------------------------------------------------------------
log_info "1. Mendaftarkan calon mahasiswa ($MHS_EMAIL)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$MHS_EMAIL\",
    \"password\": \"$MHS_PASSWORD\",
    \"password_confirmation\": \"$MHS_PASSWORD\",
    \"full_name\": \"$MHS_NAME\",
    \"phone\": \"$MHS_PHONE\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
  log_pass "Register calon mahasiswa berhasil (HTTP $HTTP_CODE)"
else
  log_fail "Register calon mahasiswa gagal (HTTP $HTTP_CODE)"
  echo -e "${YELLOW}Response Body:${NC} $BODY"
  exit 1
fi

# ------------------------------------------------------------------------------
# 2. Ambil OTP dari API Mailpit
# ------------------------------------------------------------------------------
log_info "2. Mengambil kode OTP dari Mailpit API..."
OTP=""
for i in {1..10}; do
  MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
  if [ -n "$MAIL_RESP" ]; then
    # Ambil pesan terbaru
    MSG_ID=$(echo "$MAIL_RESP" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
    if [ -z "$MSG_ID" ]; then
      MSG_ID=$(echo "$MAIL_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
    fi

    if [ -n "$MSG_ID" ]; then
      MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${MSG_ID}" || true)
      # Ekstrak 6 digit kode OTP
      EXTRACTED_OTP=$(echo "$MSG_DETAIL" | grep -oE '[0-9]{6}' | head -n1)
      if [ -n "$EXTRACTED_OTP" ]; then
        OTP="$EXTRACTED_OTP"
        break
      fi
    fi
  fi
  sleep 1
done

if [ -n "$OTP" ]; then
  log_pass "Kode OTP berhasil diperoleh dari Mailpit: $OTP"
else
  log_fail "Gagal mendapatkan kode OTP dari Mailpit API"
  echo -e "${YELLOW}Mailpit messages output:${NC} $MAIL_RESP"
  exit 1
fi

# ------------------------------------------------------------------------------
# 3. Verifikasi OTP (Verify)
# ------------------------------------------------------------------------------
log_info "3. Memverifikasi OTP calon mahasiswa..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$MHS_EMAIL\",
    \"otp\": \"$OTP\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Verifikasi OTP" "200" "$HTTP_CODE" "$BODY"

# Ekstrak access_token dan refresh_token
ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ]; then
  log_fail "Gagal mengekstrak access_token / refresh_token dari respons verify"
  exit 1
fi
log_pass "Token JWT dan Refresh Token berhasil didapatkan"

# ------------------------------------------------------------------------------
# 4. Akses Profil Calon Mahasiswa (/me)
# ------------------------------------------------------------------------------
log_info "4. Mengakses endpoint profil /me calon mahasiswa..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses /me calon mahasiswa" "200" "$HTTP_CODE" "$BODY"

# Pastikan data user sesuai
if echo "$BODY" | grep -q "$MHS_EMAIL"; then
  log_pass "Data profil /me memuat email yang valid: $MHS_EMAIL"
else
  log_fail "Data profil /me tidak memuat email $MHS_EMAIL"
  exit 1
fi

# ------------------------------------------------------------------------------
# 5. Refresh Token
# ------------------------------------------------------------------------------
log_info "5. Melakukan refresh token..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/refresh" \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Refresh token" "200" "$HTTP_CODE" "$BODY"

OLD_ACCESS_TOKEN="$ACCESS_TOKEN"
ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
  log_fail "Gagal mendapatkan access_token baru dari refresh"
  exit 1
fi
log_pass "Refresh token berhasil menghasilkan access token dan refresh token baru"

# ------------------------------------------------------------------------------
# 6. Logout
# ------------------------------------------------------------------------------
log_info "6. Melakukan logout user..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/logout" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Logout" "200" "$HTTP_CODE" "$BODY"

# ------------------------------------------------------------------------------
# 7. Akses /me dengan Token yang Sudah Logout (Harus Ditolak 401)
# ------------------------------------------------------------------------------
log_info "7. Memastikan akses /me dengan token yang telah di-logout ditolak..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses /me pasca logout ditolak" "401" "$HTTP_CODE" "$BODY"
log_pass "Akses dengan revoked token berhasil ditolak dengan HTTP 401 Unauthorized"

# ------------------------------------------------------------------------------
# 8. Login Superadmin
# ------------------------------------------------------------------------------
log_info "8. Login sebagai Superadmin..."
ADMIN_EMAIL="${SUPERADMIN_EMAIL:-admin@pmb.univ.ac.id}"
ADMIN_PASSWORD="${SUPERADMIN_PASSWORD:-AdminSecret123!}"

RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$ADMIN_EMAIL\",
    \"password\": \"$ADMIN_PASSWORD\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Login Superadmin" "200" "$HTTP_CODE" "$BODY"

ADMIN_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$ADMIN_TOKEN" ]; then
  log_fail "Gagal mendapatkan access_token superadmin"
  exit 1
fi
log_pass "Superadmin berhasil login"

# ------------------------------------------------------------------------------
# 9. Superadmin Membuat User Keuangan
# ------------------------------------------------------------------------------
KEUANGAN_EMAIL="keuangan_${RAND_ID}@univ.ac.id"
KEUANGAN_PASS="KeuanganPass123!"
log_info "9. Superadmin membuat user staf keuangan ($KEUANGAN_EMAIL)..."

RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$KEUANGAN_EMAIL\",
    \"full_name\": \"Staf Keuangan PMB\",
    \"phone\": \"081298765432\",
    \"password\": \"$KEUANGAN_PASS\",
    \"roles\": [
      {
        \"code\": \"keuangan\"
      }
    ]
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
  log_pass "Superadmin berhasil membuat user keuangan (HTTP $HTTP_CODE)"
else
  log_fail "Superadmin gagal membuat user keuangan (HTTP $HTTP_CODE)"
  echo -e "${YELLOW}Response Body:${NC} $BODY"
  exit 1
fi

# ------------------------------------------------------------------------------
# 10. Login Sebagai User Keuangan
# ------------------------------------------------------------------------------
log_info "10. Login sebagai user keuangan ($KEUANGAN_EMAIL)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$KEUANGAN_EMAIL\",
    \"password\": \"$KEUANGAN_PASS\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Login Keuangan" "200" "$HTTP_CODE" "$BODY"

KEUANGAN_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$KEUANGAN_TOKEN" ]; then
  log_fail "Gagal mendapatkan access_token keuangan"
  exit 1
fi
log_pass "User keuangan berhasil login"

# ------------------------------------------------------------------------------
# 11. User Keuangan Akses Endpoint Admin (Harus Ditolak 403 Forbidden)
# ------------------------------------------------------------------------------
log_info "11. Menguji RBAC: User keuangan mencoba mengakses endpoint admin..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/admin/users" \
  -H "Authorization: Bearer $KEUANGAN_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses admin oleh user non-superadmin ditolak" "403" "$HTTP_CODE" "$BODY"
log_pass "RBAC Gateway berhasil memblokir akses ke /admin/users dengan HTTP 403 Forbidden"

# ------------------------------------------------------------------------------
# Ringkasan Akhir
# ------------------------------------------------------------------------------
echo "------------------------------------------------------------------------------"
echo -e "${GREEN}SEMUA PENGUJIAN END-TO-END BERHASIL! (Total PASS: $pass_count, FAIL: $fail_count)${NC}"
echo "=============================================================================="
exit 0
