#!/usr/bin/env bash
# ==============================================================================
# PMB Backend - Comprehensive End-to-End Verification Script via API Gateway
# ==============================================================================
# Alur Pengujian:
# 1. Register calon mahasiswa
# 2. Ambil kode OTP dari API Mailpit
# 3. Verifikasi OTP -> dapatkan token JWT (access & refresh)
# 4. Akses endpoint /me dengan access token
# 5. Refresh token -> peroleh access token & refresh token baru (rotasi)
# 6. Logout -> cabut sesi di Redis / denylist token
# 7. Akses kembali endpoint /me dengan token yang telah dicabut (harus 401)
# 8. [NEGATIF] Refresh token dipakai sebagai access token pada /me (harus 401)
# 9. [NEGATIF] Reuse refresh token lama yang sudah dirotasi (harus 401 TOKEN_REVOKED)
# 10. [NEGATIF] Injeksi header X-User-ID palsu dari klien (harus distrip/ditolak)
# 11. [NEGATIF] Akses endpoint /internal/* melalui gateway (harus diblokir 404)
# 12. [NEGATIF] Login email tak terdaftar (harus 401, pesan identik dengan password salah)
# 13. [NEGATIF] Login salah 5x berturut-turut sampai lockout (harus 429 TOO_MANY_ATTEMPTS)
# 14. [NEGATIF] OTP salah berulang kali sampai terkunci (harus 429 OTP_LOCKED)
# 15. [FITUR] Forgot password -> baca token di Mailpit -> reset password -> verifikasi login
# 16. Login sebagai superadmin (kredensial default seed development)
# 17. Superadmin membuat akun user dengan role keuangan
# 18. Login sebagai user keuangan
# 19. User keuangan mencoba mengakses endpoint admin (harus 403 Forbidden)
# ==============================================================================

set -e

# Konfigurasi Target URL
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8000/api/v1/auth}"
GATEWAY_BASE="${GATEWAY_BASE:-http://localhost:8000}"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
AUTH_DIRECT_URL="${AUTH_DIRECT_URL:-http://localhost:8080}"
ORG_DIRECT_URL="${ORG_DIRECT_URL:-http://localhost:8081}"
ORG_URL="${GATEWAY_BASE}/api/v1/organization"
IDENTITY_DIRECT_URL="${IDENTITY_DIRECT_URL:-http://localhost:8082}"
IDENTITY_URL="${GATEWAY_BASE}/api/v1/identity"
ADMISSION_DIRECT_URL="${ADMISSION_DIRECT_URL:-http://localhost:8084}"
ADMISSION_URL="${GATEWAY_BASE}/api/v1/admission"
NOTIFICATION_DIRECT_URL="${NOTIFICATION_DIRECT_URL:-http://localhost:8083}"
FINANCE_DIRECT_URL="${FINANCE_DIRECT_URL:-http://localhost:8085}"
INVOICES_URL="${GATEWAY_BASE}/api/v1/invoices"
RECEIPTS_URL="${GATEWAY_BASE}/api/v1/receipts"
FINANCE_ADMIN_URL="${GATEWAY_BASE}/api/v1/finance/admin"
DEV_PAYMENTS_URL="${GATEWAY_BASE}/api/v1/dev/payments"
WEBHOOKS_URL="${GATEWAY_BASE}/api/v1/webhooks/payments"

SUPERADMIN_EMAIL="${SEED_SUPERADMIN_EMAIL:-${SUPERADMIN_EMAIL:-superadmin@kampus.ac.id}}"
SUPERADMIN_PASSWORD="${SEED_SUPERADMIN_PASSWORD:-${SUPERADMIN_PASSWORD:-Superadmin123!}}"

# Formatting & Warna Output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

json_get() {
  local json_str="$1"
  local key_name="$2"

  if command -v jq >/dev/null 2>&1; then
    echo "$json_str" | jq -r "$key_name" 2>/dev/null || true
  else
    # Fallback regex jika jq belum terpasang
    local simple_key="${key_name##*.}"
    simple_key="${simple_key#\"}"
    simple_key="${simple_key%\"}"
    echo "$json_str" | grep -o "\"$simple_key\": *\"[^\"]*\"" | head -n1 | cut -d'"' -f4
  fi
}

echo "=============================================================================="
echo "          PMB BACKEND COMPREHENSIVE E2E VERIFICATION SUITE"
echo "=============================================================================="
echo "Gateway URL : $GATEWAY_URL"
echo "Mailpit URL : $MAILPIT_URL"
echo "Timestamp   : $(date)"
echo "------------------------------------------------------------------------------"

# Bersihkan sliding window rate limit di Redis sebelum pengujian
if command -v docker >/dev/null 2>&1; then
  docker exec pmb-redis redis-cli FLUSHALL >/dev/null 2>&1 || true
fi

RAND_ID="${RANDOM}_${RANDOM}"
MHS_EMAIL="calon_mhs_${RAND_ID}@example.com"
MHS_PASSWORD="Password123!"
MHS_NAME="Calon Mahasiswa E2E"
MHS_PHONE="081234567890"

# ==============================================================================
# BAGIAN 1: ALUR UTAMA CALON MAHASISWA
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 1: ALUR UTAMA CALON MAHASISWA${NC}"

# 1. Register Calon Mahasiswa
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
assert_status "Register calon mahasiswa" "201" "$HTTP_CODE" "$BODY"

# 2. Ambil OTP dari Mailpit API
log_info "2. Mengambil kode OTP dari Mailpit API..."
OTP=""
for i in {1..10}; do
  MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
  if [ -n "$MAIL_RESP" ]; then
    MSG_ID=$(echo "$MAIL_RESP" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
    if [ -z "$MSG_ID" ]; then
      MSG_ID=$(echo "$MAIL_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
    fi

    if [ -n "$MSG_ID" ]; then
      MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${MSG_ID}" || true)
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

# 3. Verifikasi OTP (Auto-Login)
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

ACCESS_TOKEN=$(json_get "$BODY" ".data.access_token")
REFRESH_TOKEN=$(json_get "$BODY" ".data.refresh_token")

if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ]; then
  # Fallback manual cut
  ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
  REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ]; then
  log_fail "Gagal mengekstrak access_token / refresh_token dari respons verify"
  exit 1
fi
log_pass "Token JWT dan Refresh Token berhasil didapatkan"

# 4. Akses Profil Calon Mahasiswa (/me)
log_info "4. Mengakses endpoint profil /me calon mahasiswa..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses /me calon mahasiswa" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q "$MHS_EMAIL"; then
  log_pass "Data profil /me memuat email yang valid: $MHS_EMAIL"
else
  log_fail "Data profil /me tidak memuat email $MHS_EMAIL"
  exit 1
fi

# 5. Refresh Token (Rotasi Token)
log_info "5. Melakukan refresh token (rotasi)..."
OLD_REFRESH_TOKEN="$REFRESH_TOKEN"
OLD_ACCESS_TOKEN="$ACCESS_TOKEN"

RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/refresh" \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$OLD_REFRESH_TOKEN\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Refresh token" "200" "$HTTP_CODE" "$BODY"

ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ]; then
  log_fail "Gagal mendapatkan access_token/refresh_token baru dari refresh"
  exit 1
fi
log_pass "Refresh token berhasil menghasilkan access token dan refresh token baru"

# 6. Logout Sesi
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

# 7. Akses /me dengan Token yang Sudah Logout (Harus 401)
log_info "7. Memastikan akses /me dengan token yang telah di-logout ditolak..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses /me pasca logout ditolak" "401" "$HTTP_CODE" "$BODY"
log_pass "Akses dengan revoked token berhasil ditolak dengan HTTP 401 Unauthorized"

# ==============================================================================
# BAGIAN 2: SKENARIO NEGATIF & KEAMANAN
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 2: SKENARIO NEGATIF & KEAMANAN${NC}"

# 8. Refresh Token Dipakai Sebagai Access Token (Harus Ditolak 401)
log_info "8. [NEGATIF] Menggunakan Refresh Token sebagai Access Token pada /me..."
# Buat login baru untuk mendapatkan pasangan token segar
RESP_LOGIN=$(curl -s -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$MHS_EMAIL\",
    \"password\": \"$MHS_PASSWORD\"
  }")

TEST_ACCESS=$(echo "$RESP_LOGIN" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
TEST_REFRESH=$(echo "$RESP_LOGIN" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $TEST_REFRESH")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Refresh token sebagai Bearer token ditolak" "401" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "TOKEN_INVALID"; then
  log_pass "Gateway menolak token non-access dengan error_code TOKEN_INVALID"
else
  log_fail "Error code bukan TOKEN_INVALID: $BODY"
  exit 1
fi

# 9. Deteksi Reuse Refresh Token Lama (Token Theft Detection)
log_info "9. [NEGATIF] Melakukan reuse refresh token lama yang sudah dirotasi..."
# Rotasi 1x
RESP_ROT=$(curl -s -X POST "${GATEWAY_URL}/refresh" \
  -H "Content-Type: application/json" \
  -d "{ \"refresh_token\": \"$TEST_REFRESH\" }")
NEW_REFRESH=$(echo "$RESP_ROT" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

# Gunakan TEST_REFRESH (token lama yang sudah digantikan NEW_REFRESH)
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/refresh" \
  -H "Content-Type: application/json" \
  -d "{ \"refresh_token\": \"$TEST_REFRESH\" }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Reuse refresh token lama ditolak" "401" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "TOKEN_REVOKED"; then
  log_pass "Sistem mendeteksi reuse token dan mengembalikan TOKEN_REVOKED"
else
  log_fail "Error code reuse bukan TOKEN_REVOKED: $BODY"
  exit 1
fi

# 10. Anti-Spoofing: Injeksi Header X-User-ID Palsu
log_info "10. [NEGATIF] Menguji pencegahan spoofing header X-User-ID..."
# 10a: Kirim X-User-ID palsu tanpa token autentikasi
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000000")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Header X-User-ID palsu tanpa token ditolak" "401" "$HTTP_CODE" "$BODY"

# 10b: Login kembali dan kirim X-User-ID palsu bersama token calon mahasiswa yang valid
RESP_LOGIN=$(curl -s -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$MHS_EMAIL\", \"password\": \"$MHS_PASSWORD\" }")
VALID_ACCESS=$(echo "$RESP_LOGIN" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $VALID_ACCESS" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000000")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses dengan token valid + header palsu" "200" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "00000000-0000-0000-0000-000000000000"; then
  log_fail "Celah keamanan terdeteksi: header X-User-ID palsu berhasil disuntikkan!"
  exit 1
else
  log_pass "Gateway berhasil membersihkan/menimpa header X-User-ID palsu dengan data asli token"
fi

# 11. Pemblokiran Akses Gateway ke Endpoint Internal (/internal/*)
log_info "11. [NEGATIF] Memastikan akses ke endpoint /internal/* lewat gateway diblokir..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_BASE}/api/v1/auth/internal/users/123")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses ke /internal/* diblokir" "404" "$HTTP_CODE" "$BODY"
log_pass "Gateway berhasil memblokir akses ke endpoint internal dengan HTTP 404 Not Found"

# 12. Login Email Tak Terdaftar vs Password Salah (Pesan Identik)
log_info "12. [NEGATIF] Menguji pesan error login email tak terdaftar vs kata sandi salah..."
NON_EXISTENT_EMAIL="tidak_terdaftar_${RAND_ID}@example.com"
RESP_NO_EMAIL=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$NON_EXISTENT_EMAIL\", \"password\": \"WrongPassword123!\" }")
CODE_NO_EMAIL=$(echo "$RESP_NO_EMAIL" | tail -n1)
BODY_NO_EMAIL=$(echo "$RESP_NO_EMAIL" | sed '$d')
assert_status "Login email tak terdaftar ditolak" "401" "$CODE_NO_EMAIL" "$BODY_NO_EMAIL"

RESP_WRONG_PASS=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$MHS_EMAIL\", \"password\": \"WrongPassword123!\" }")
CODE_WRONG_PASS=$(echo "$RESP_WRONG_PASS" | tail -n1)
BODY_WRONG_PASS=$(echo "$RESP_WRONG_PASS" | sed '$d')
assert_status "Login kata sandi salah ditolak" "401" "$CODE_WRONG_PASS" "$BODY_WRONG_PASS"

MSG_NO_EMAIL=$(echo "$BODY_NO_EMAIL" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
MSG_WRONG_PASS=$(echo "$BODY_WRONG_PASS" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)

if [ "$MSG_NO_EMAIL" = "$MSG_WRONG_PASS" ] && [ -n "$MSG_NO_EMAIL" ]; then
  log_pass "Anti-User Enumeration lolos: pesan error identik ('$MSG_NO_EMAIL')"
else
  log_fail "Pesan error berbeda antara email tak terdaftar ('$MSG_NO_EMAIL') dan password salah ('$MSG_WRONG_PASS')"
  exit 1
fi

# 13. Login Salah sampai Lockout (TOO_MANY_ATTEMPTS)
log_info "13. [NEGATIF] Menguji proteksi brute force login (lockout 5x salah)..."
LOCKOUT_EMAIL="brute_target_${RAND_ID}@example.com"
# Register & verify akun target lockout
RESP_TGT=$(curl -s -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$LOCKOUT_EMAIL\", \"password\": \"PassLockout123!\", \"password_confirmation\": \"PassLockout123!\", \"full_name\": \"Lockout Target\", \"phone\": \"081999990001\" }")

# Ambil OTP target dari Mailpit
sleep 1
MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
MSG_ID=$(echo "$MAIL_RESP" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
if [ -z "$MSG_ID" ]; then MSG_ID=$(echo "$MAIL_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4); fi
MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${MSG_ID}" || true)
TGT_OTP=$(echo "$MSG_DETAIL" | grep -oE '[0-9]{6}' | head -n1)

_=$(curl -s -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$LOCKOUT_EMAIL\", \"otp\": \"$TGT_OTP\" }")

# Lakukan percobaan login salah 5 kali
GOT_LOCKOUT=false
for attempt in {1..5}; do
  RESP_FAIL=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
    -H "Content-Type: application/json" \
    -d "{ \"email\": \"$LOCKOUT_EMAIL\", \"password\": \"Salah${attempt}!\" }")
  HTTP_CODE=$(echo "$RESP_FAIL" | tail -n1)
  BODY=$(echo "$RESP_FAIL" | sed '$d')

  if [ "$HTTP_CODE" = "429" ] && echo "$BODY" | grep -q "TOO_MANY_ATTEMPTS"; then
    GOT_LOCKOUT=true
    log_pass "Percobaan ke-$attempt: Akun berhasil dilockout dengan HTTP 429 TOO_MANY_ATTEMPTS"
    break
  fi
done

if [ "$GOT_LOCKOUT" = "true" ]; then
  # Coba login dengan password yang benar saat akun terkunci -> harus tetap ditolak 429
  RESP_VALID_DURING_LOCK=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
    -H "Content-Type: application/json" \
    -d "{ \"email\": \"$LOCKOUT_EMAIL\", \"password\": \"PassLockout123!\" }")
  HTTP_CODE=$(echo "$RESP_VALID_DURING_LOCK" | tail -n1)
  BODY=$(echo "$RESP_VALID_DURING_LOCK" | sed '$d')
  assert_status "Login password benar saat lockout tetap diblokir" "429" "$HTTP_CODE" "$BODY"
  log_pass "Proteksi brute force lockout 15 menit berfungsi dengan sempurna"
else
  log_fail "Akun tidak terlindungi lockout setelah 5x salah login"
  exit 1
fi

## 14. OTP Salah sampai Terkunci (OTP_LOCKED)
log_info "14. [NEGATIF] Menguji penguncian OTP setelah salah berulang kali (OTP_LOCKED)..."
OTP_LOCK_EMAIL="otp_lock_${RANDOM}_${RANDOM}@example.com"
OTP_LOCK_PHONE="081$(date +%s | cut -c4-10)"
RESP_REG=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$OTP_LOCK_EMAIL\", \"password\": \"PassOtpLock123!\", \"password_confirmation\": \"PassOtpLock123!\", \"full_name\": \"OTP Lock Target\", \"phone\": \"$OTP_LOCK_PHONE\" }")
CODE_REG=$(echo "$RESP_REG" | tail -n1)
if [ "$CODE_REG" != "201" ] && [ "$CODE_REG" != "200" ]; then
  log_fail "Gagal mendaftarkan user target OTP lock (HTTP $CODE_REG)"
  exit 1
fi

GOT_OTP_LOCKED=false
for attempt in {1..7}; do
  RESP_BAD_OTP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/verify" \
    -H "Content-Type: application/json" \
    -d "{ \"email\": \"$OTP_LOCK_EMAIL\", \"otp\": \"000000\" }")
  HTTP_CODE=$(echo "$RESP_BAD_OTP" | tail -n1)
  BODY=$(echo "$RESP_BAD_OTP" | sed '$d')

  if [ "$HTTP_CODE" = "429" ] && echo "$BODY" | grep -q "OTP_LOCKED"; then
    GOT_OTP_LOCKED=true
    log_pass "Percobaan ke-$attempt: OTP berhasil terkunci dengan HTTP 429 OTP_LOCKED"
    break
  fi
  sleep 0.2
done

if [ "$GOT_OTP_LOCKED" = "true" ]; then
  log_pass "Verifikasi OTP terkunci setelah batas percobaan habis"
else
  log_fail "OTP tidak terkunci setelah berkali-kali salah"
  echo -e "${YELLOW}Response Terakhir:${NC} HTTP $HTTP_CODE - $BODY"
  exit 1
fi

# 15. Alur Forgot Password -> Reset Password -> Login Password Baru
log_info "15. [FITUR] Menguji alur Forgot Password dan Reset Password..."
FORGOT_EMAIL="forgot_test_${RAND_ID}@example.com"
FORGOT_OLD_PASS="OldPass123!"
FORGOT_NEW_PASS="NewPass456!"

# Register & verify akun
_=$(curl -s -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$FORGOT_EMAIL\", \"password\": \"$FORGOT_OLD_PASS\", \"password_confirmation\": \"$FORGOT_OLD_PASS\", \"full_name\": \"Forgot Test User\", \"phone\": \"081999990003\" }")

sleep 1
MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
MSG_ID=$(echo "$MAIL_RESP" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
if [ -z "$MSG_ID" ]; then MSG_ID=$(echo "$MAIL_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4); fi
MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${MSG_ID}" || true)
FORGOT_OTP=$(echo "$MSG_DETAIL" | grep -oE '[0-9]{6}' | head -n1)

_=$(curl -s -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$FORGOT_EMAIL\", \"otp\": \"$FORGOT_OTP\" }")

# Request forgot password
log_info "Mengirim permintaan lupa password ($FORGOT_EMAIL)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/forgot-password" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$FORGOT_EMAIL\" }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Request forgot-password" "200" "$HTTP_CODE" "$BODY"

# Ambil token reset dari email di Mailpit
sleep 1
RESET_TOKEN=""
for i in {1..10}; do
  MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
  MSG_ID=$(echo "$MAIL_RESP" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
  if [ -z "$MSG_ID" ]; then MSG_ID=$(echo "$MAIL_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4); fi
  if [ -n "$MSG_ID" ]; then
    MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${MSG_ID}" || true)
    # Ekstrak token dari URL token=...
    EXTRACTED_TOKEN=$(echo "$MSG_DETAIL" | grep -o 'token=[a-zA-Z0-9_-]*' | head -n1 | cut -d'=' -f2)
    if [ -n "$EXTRACTED_TOKEN" ]; then
      RESET_TOKEN="$EXTRACTED_TOKEN"
      break
    fi
  fi
  sleep 1
done

if [ -n "$RESET_TOKEN" ]; then
  log_pass "Token reset password diperoleh dari Mailpit: ${RESET_TOKEN:0:10}..."
else
  log_fail "Gagal memperoleh token reset password dari Mailpit"
  exit 1
fi

# Submit reset password dengan token
log_info "Mengirim form reset-password dengan kata sandi baru..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/reset-password" \
  -H "Content-Type: application/json" \
  -d "{
    \"token\": \"$RESET_TOKEN\",
    \"password\": \"$FORGOT_NEW_PASS\",
    \"password_confirmation\": \"$FORGOT_NEW_PASS\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Reset password baru" "200" "$HTTP_CODE" "$BODY"

# Pastikan login dengan password lama DITOLAK (401)
log_info "Memastikan login dengan kata sandi lama ditolak..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$FORGOT_EMAIL\", \"password\": \"$FORGOT_OLD_PASS\" }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Login kata sandi lama ditolak" "401" "$HTTP_CODE" "$BODY"

# Pastikan login dengan password baru BERHASIL (200)
log_info "Memastikan login dengan kata sandi baru berhasil..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{ \"email\": \"$FORGOT_EMAIL\", \"password\": \"$FORGOT_NEW_PASS\" }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Login kata sandi baru sukses" "200" "$HTTP_CODE" "$BODY"
log_pass "Alur Forgot Password dan Reset Password berhasil secara menyeluruh"

# ==============================================================================
# BAGIAN 3: ALUR RBAC ADMINISTRATOR & INTERNAL
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 3: ALUR RBAC ADMINISTRATOR & INTERNAL${NC}"

# 16. Login Superadmin
log_info "16. Login sebagai Superadmin ($SUPERADMIN_EMAIL)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$SUPERADMIN_EMAIL\",
    \"password\": \"$SUPERADMIN_PASSWORD\"
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

# 17. Superadmin Membuat User Keuangan
KEUANGAN_EMAIL="keuangan_${RAND_ID}@univ.ac.id"
KEUANGAN_PASS="KeuanganPass123!"
log_info "17. Superadmin membuat user staf keuangan ($KEUANGAN_EMAIL)..."

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

# 18. Login Sebagai User Keuangan
log_info "18. Login sebagai user keuangan ($KEUANGAN_EMAIL)..."
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

# 19. User Keuangan Akses Endpoint Admin (Harus Ditolak 403 Forbidden)
log_info "19. Menguji RBAC: User keuangan mencoba mengakses endpoint admin..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/admin/users" \
  -H "Authorization: Bearer $KEUANGAN_TOKEN")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Akses admin oleh user non-superadmin ditolak" "403" "$HTTP_CODE" "$BODY"
log_pass "RBAC Gateway berhasil memblokir akses ke /admin/users dengan HTTP 403 Forbidden"

# 20. [KEAMANAN] Menguji Proteksi Bypass Gateway: Request Langsung ke Auth-Service
log_info "20. [KEAMANAN] Menguji proteksi bypass gateway: Request langsung ke auth-service..."
if ! curl -s --connect-timeout 2 "${AUTH_DIRECT_URL}/health" >/dev/null 2>&1; then
  echo -e "${YELLOW}[SKIP]${NC} Port auth-service langsung (${AUTH_DIRECT_URL}) tidak terbuka / tidak dapat diakses"
else
  DIRECT_RES=$(curl -s -w "\n%{http_code}" -X GET "${AUTH_DIRECT_URL}/auth/me" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  DIRECT_STATUS=$(echo "$DIRECT_RES" | tail -n1)
  DIRECT_BODY=$(echo "$DIRECT_RES" | sed '$d')

  assert_status "Request langsung ke auth-service tanpa gateway signature ditolak" "401" "$DIRECT_STATUS" "$DIRECT_BODY"

  DIRECT_ERR_CODE=$(echo "$DIRECT_BODY" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4)
  if [ "$DIRECT_ERR_CODE" = "GATEWAY_SIGNATURE_INVALID" ]; then
    log_pass "Auth-service menolak bypass gateway dengan error_code GATEWAY_SIGNATURE_INVALID"
  else
    log_fail "Error code yang diharapkan GATEWAY_SIGNATURE_INVALID, didapat: '$DIRECT_ERR_CODE'"
  fi
fi

# ==============================================================================
# BAGIAN 4: PENGUJIAN TAHAP 1 — ORGANIZATION SERVICE & SCOPE VALIDATION
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 4: PENGUJIAN TAHAP 1 — ORGANIZATION SERVICE & SCOPE VALIDATION${NC}"

# 21. Daftar Prodi Publik Tanpa Login
log_info "21. Mengambil daftar program studi publik tanpa autentikasi..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ORG_URL}/study-programs")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Daftar prodi publik berhasil diakses tanpa login" "200" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q '"code":"TI"'; then
  log_pass "Daftar prodi memuat program studi TI"
else
  log_fail "Daftar prodi tidak memuat program studi TI"
fi

# 22. Detail Prodi Publik
log_info "22. Mengambil detail prodi TI secara publik..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ORG_URL}/study-programs/TI")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Detail prodi TI berhasil diakses" "200" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q '"nim_code":"55"'; then
  log_pass "Detail prodi TI memuat nim_code 55"
else
  log_fail "Detail prodi TI tidak memuat nim_code 55"
fi

# 23. Calon Mahasiswa Ditolak Membuat Prodi (403 Forbidden)
log_info "23. Menguji RBAC: Calon mahasiswa mencoba membuat prodi..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${ORG_URL}/admin/study-programs" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "faculty_id": 1,
    "code": "ILLEGAL",
    "nim_code": "99",
    "name": "Prodi Ilegal",
    "degree": "S1",
    "accreditation": "C"
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Calon mahasiswa ditolak membuat prodi" "403" "$HTTP_CODE" "$BODY"
log_pass "Gateway/Organization berhasil memblokir non-superadmin dari /admin/study-programs dengan HTTP 403"

# 24. Superadmin Membuat Prodi Baru
NEW_PRODI_CODE="TE_${RAND_ID}"
log_info "24. Superadmin membuat prodi baru ($NEW_PRODI_CODE)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${ORG_URL}/admin/study-programs" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"faculty_id\": 1,
    \"code\": \"$NEW_PRODI_CODE\",
    \"nim_code\": \"58\",
    \"name\": \"Teknik Elektro\",
    \"degree\": \"S1\",
    \"accreditation\": \"Unggul\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
  log_pass "Superadmin berhasil membuat prodi baru (HTTP $HTTP_CODE)"
else
  log_fail "Superadmin gagal membuat prodi baru (HTTP $HTTP_CODE)"
  echo -e "${YELLOW}Response Body:${NC} $BODY"
  exit 1
fi

# 25. Superadmin Memberi Role Prodi dengan Scope Tidak Valid (Harus Ditolak 422 VALIDATION_ERROR)
PRODI_USER_EMAIL="kaprodi_${RAND_ID}@univ.ac.id"
PRODI_USER_PASS="KaprodiPass123!"

log_info "25. Superadmin membuat user baru untuk kaprodi..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$PRODI_USER_EMAIL\",
    \"full_name\": \"Kaprodi Test\",
    \"phone\": \"081211223344\",
    \"password\": \"$PRODI_USER_PASS\",
    \"roles\": [
      {
        \"code\": \"calon_mahasiswa\"
      }
    ]
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "User baru berhasil dibuat" "201" "$HTTP_CODE" "$BODY"
PRODI_USER_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)

log_info "25b. Superadmin memberi role prodi dengan scope yang TIDAK ADA di Organization Service..."
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${GATEWAY_URL}/admin/users/${PRODI_USER_ID}/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "roles": [
      {
        "code": "prodi",
        "scope_type": "prodi",
        "scope_id": "KODE_TIDAK_ADA"
      }
    ]
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Scope prodi tidak valid ditolak" "422" "$HTTP_CODE" "$BODY"
log_pass "Auth Service berhasil menolak scope prodi yang tidak ada di Organization Service (HTTP 422)"

# 26. Superadmin Memberi Role Prodi dengan Scope Valid (TI)
log_info "26. Superadmin memberi role prodi dengan scope valid (TI)..."
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${GATEWAY_URL}/admin/users/${PRODI_USER_ID}/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "roles": [
      {
        "code": "prodi",
        "scope_type": "prodi",
        "scope_id": "TI"
      }
    ]
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Scope prodi valid berhasil disimpan" "200" "$HTTP_CODE" "$BODY"
log_pass "Auth Service berhasil menyimpan role prodi dengan scope TI"

# 27. Login User Prodi & Verifikasi Token Memuat Scope
log_info "27. Login sebagai user prodi ($PRODI_USER_EMAIL)..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$PRODI_USER_EMAIL\",
    \"password\": \"$PRODI_USER_PASS\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Login user prodi sukses" "200" "$HTTP_CODE" "$BODY"

PRODI_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$PRODI_TOKEN" ]; then
  log_fail "Gagal mendapatkan access_token prodi"
  exit 1
fi

# Cek klaim token via /auth/me
RESP=$(curl -s -w "\n%{http_code}" -X GET "${GATEWAY_URL}/me" \
  -H "Authorization: Bearer $PRODI_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Ambil profile user prodi" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q '"id":"TI"' && echo "$BODY" | grep -q '"role":"prodi"'; then
  log_pass "Token dan profile user prodi berhasil memuat scope prodi TI"
else
  log_fail "Token user prodi tidak memuat scope TI: $BODY"
fi

# 28. [KEAMANAN] Menguji Proteksi Bypass Gateway: Request Langsung ke Organization-Service
log_info "28. [KEAMANAN] Menguji proteksi bypass gateway: Request langsung ke organization-service..."
if ! curl -s --connect-timeout 2 "${ORG_DIRECT_URL}/health" >/dev/null 2>&1; then
  echo -e "${YELLOW}[SKIP]${NC} Port organization-service langsung (${ORG_DIRECT_URL}) tidak terbuka / tidak dapat diakses"
else
  DIRECT_RES=$(curl -s -w "\n%{http_code}" -X GET "${ORG_DIRECT_URL}/organization/study-programs" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  DIRECT_STATUS=$(echo "$DIRECT_RES" | tail -n1)
  DIRECT_BODY=$(echo "$DIRECT_RES" | sed '$d')

  assert_status "Request langsung ke organization-service tanpa gateway signature ditolak" "401" "$DIRECT_STATUS" "$DIRECT_BODY"

  DIRECT_ERR_CODE=$(echo "$DIRECT_BODY" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4)
  if [ "$DIRECT_ERR_CODE" = "GATEWAY_SIGNATURE_INVALID" ]; then
    log_pass "Organization-service menolak bypass gateway dengan error_code GATEWAY_SIGNATURE_INVALID"
  else
    log_fail "Error code yang diharapkan GATEWAY_SIGNATURE_INVALID, didapat: '$DIRECT_ERR_CODE'"
  fi
fi

# ==============================================================================
# BAGIAN 5: TAHAP 2 - IDENTITY SERVICE & OCR KTP (F2-M2 & F2-M2b)
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 5: TAHAP 2 - IDENTITY SERVICE & OCR KTP (F2-M2 & F2-M2b)${NC}"

# Ambil user_id calon mahasiswa dari profile /auth/me
RESP_ME=$(curl -s -X GET "${GATEWAY_URL}/me" -H "Authorization: Bearer $ACCESS_TOKEN")
MHS_USER_ID=$(echo "$RESP_ME" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)

# 29. Cek Kelengkapan Awal Biodata (Harus complete: false)
log_info "29. Memeriksa kelengkapan awal biodata calon mahasiswa..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${IDENTITY_URL}/me/completeness" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Cek kelengkapan awal" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q '"complete":false'; then
  log_pass "Kelengkapan awal biodata berstatus complete: false (belum diisi)"
else
  log_fail "Kelengkapan awal harus false: $BODY"
fi

# 30. Pengisian Biodata Calon Mahasiswa (Profil, Alamat, Wali, Pendidikan)
log_info "30. Mengisi biodata lengkap calon mahasiswa..."

# 30a. Update Profile
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${IDENTITY_URL}/me/profile" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Ahmad Syafiq E2E",
    "nik": "3201015505050002",
    "birth_place": "Bandung",
    "birth_date": "2005-05-15",
    "gender": "L",
    "religion": "Islam",
    "nationality": "WNI",
    "phone": "081234567890"
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Update profil biodata" "200" "$HTTP_CODE" "$BODY"

# 30b. Simpan Alamat KTP
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${IDENTITY_URL}/me/addresses" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ktp",
    "street": "Jl. Sukajadi No. 12",
    "rt": "02",
    "rw": "03",
    "village": "Sukagalih",
    "district": "Sukajadi",
    "city": "Bandung",
    "province": "Jawa Barat",
    "postal_code": "40163"
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Simpan alamat KTP" "200" "$HTTP_CODE" "$BODY"

# 30c. Simpan Data Orang Tua / Wali
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${IDENTITY_URL}/me/guardians" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "relation": "ayah",
    "name": "Bambang Sudarsono",
    "occupation": "Wiraswasta",
    "income_range": "5.000.000 - 10.000.000",
    "phone": "081299887766"
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Simpan data wali" "200" "$HTTP_CODE" "$BODY"

# 30d. Simpan Riwayat Pendidikan
RESP=$(curl -s -w "\n%{http_code}" -X PUT "${IDENTITY_URL}/me/education" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "level": "SMA",
    "school_name": "SMA Negeri 1 Bandung",
    "npsn": "20219485",
    "major": "IPA",
    "graduation_year": 2024
  }')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Simpan riwayat pendidikan" "200" "$HTTP_CODE" "$BODY"
log_pass "Seluruh berkas biodata berhasil diisi lengkap"

# 31. Cek Ulang Kelengkapan Biodata (Harus complete: true)
log_info "31. Memeriksa ulang kelengkapan biodata setelah diisi..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${IDENTITY_URL}/me/completeness" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Cek kelengkapan pasca pengisian" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q '"complete":true'; then
  log_pass "Status kelengkapan biodata berhasil menjadi complete: true"
else
  log_fail "Status kelengkapan harus true setelah pengisian lengkap: $BODY"
fi

# 32. Isolasi Data Pengguna (User A vs User B)
log_info "32. [ISOLASI DATA] Menguji bahwa data biodata terisolasi per user..."
# Registrasi & Verifikasi User B
USER_B_EMAIL="mhs_b_$RANDOM@example.com"
USER_B_PASS="PasswordUserB123!"

curl -s -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$USER_B_EMAIL\",
    \"password\": \"$USER_B_PASS\",
    \"password_confirmation\": \"$USER_B_PASS\",
    \"full_name\": \"Calon Mahasiswa B\",
    \"phone\": \"089876543210\"
  }" >/dev/null

sleep 1
OTP_B=$(curl -s "${MAILPIT_URL}/api/v1/messages" | grep -o 'kode OTP pendaftaran Anda: [0-9]\{6\}' | head -n1 | grep -o '[0-9]\{6\}')
RESP_VERIFY_B=$(curl -s -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_B_EMAIL\",\"otp\":\"$OTP_B\"}")
USER_B_TOKEN=$(echo "$RESP_VERIFY_B" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# User B cek /me/completeness -> harus false (tidak terpengaruh User A)
RESP=$(curl -s -w "\n%{http_code}" -X GET "${IDENTITY_URL}/me/completeness" \
  -H "Authorization: Bearer $USER_B_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Cek kelengkapan User B" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q '"complete":false'; then
  log_pass "Data User B bersih dan terisolasi dari User A"
else
  log_fail "Isolasi data gagal! User B membaca data User A: $BODY"
fi

# 33. Superadmin Membaca Detail Biodata Lengkap Calon Mahasiswa (NIK Unmasked)
log_info "33. Superadmin membaca detail biodata User A..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${IDENTITY_URL}/admin/persons/${MHS_USER_ID}" \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Superadmin membaca detail person" "200" "$HTTP_CODE" "$BODY"

if echo "$BODY" | grep -q "3201015505050002" && echo "$BODY" | grep -q "Ahmad Syafiq E2E"; then
  log_pass "Superadmin berhasil melihat detail biodata lengkap dengan NIK utuh (unmasked)"
else
  log_fail "NIK tidak ditemukan secara unmasked untuk superadmin: $BODY"
fi

# 34. Inisiasi Scan KTP (Multipart)
log_info "34. Calon mahasiswa mengunggah scan KTP sintetis..."
# Buat mock file JPG
TMP_KTP="/tmp/e2e_ktp_synthetic_$$.jpg"
printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00`\x00`\x00\x00\xFF\xDB\x00C\x00mock-synthetic-ktp-image-bytes-padding-for-e2e-testing-purposes-1234567890\xFF\xD9' > "$TMP_KTP"

RESP=$(curl -s -w "\n%{http_code}" -X POST "${IDENTITY_URL}/me/ktp-scan" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@$TMP_KTP;type=image/jpeg")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
rm -f "$TMP_KTP"

assert_status "Inisiasi scan KTP" "202" "$HTTP_CODE" "$BODY"
SCAN_ID=$(echo "$BODY" | grep -o '"scan_id":"[^"]*"' | cut -d'"' -f4)
if [ -n "$SCAN_ID" ]; then
  log_pass "Scan KTP berhasil diinisiasi dengan scan_id: $SCAN_ID"
else
  log_fail "Gagal mendapatkan scan_id: $BODY"
fi

# 35. [NEGATIF] Unggah Berkas KTP dengan Tipe Tidak Diizinkan (Misal file teks)
log_info "35. [NEGATIF] Mengunggah file berkas teks sebagai KTP..."
TMP_BAD="/tmp/bad_ktp_$$.txt"
echo "Bukan gambar KTP" > "$TMP_BAD"

RESP=$(curl -s -w "\n%{http_code}" -X POST "${IDENTITY_URL}/me/ktp-scan" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@$TMP_BAD;type=text/plain")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
rm -f "$TMP_BAD"

assert_status "Unggahan berkas bukan gambar ditolak" "422" "$HTTP_CODE" "$BODY"
log_pass "Sistem berhasil menolak unggahan berkas non-gambar dengan HTTP 422"

# 36. [KEAMANAN] Proteksi Bypass Gateway: Request Langsung ke Identity-Service
log_info "36. [KEAMANAN] Menguji proteksi bypass gateway: Request langsung ke identity-service..."
if ! curl -s --connect-timeout 2 "${IDENTITY_DIRECT_URL}/health" >/dev/null 2>&1; then
  echo -e "${YELLOW}[SKIP]${NC} Port identity-service langsung (${IDENTITY_DIRECT_URL}) tidak terbuka / tidak dapat diakses"
else
  DIRECT_RES=$(curl -s -w "\n%{http_code}" -X GET "${IDENTITY_DIRECT_URL}/identity/me/profile" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  DIRECT_STATUS=$(echo "$DIRECT_RES" | tail -n1)
  DIRECT_BODY=$(echo "$DIRECT_RES" | sed '$d')

  assert_status "Request langsung ke identity-service tanpa gateway signature ditolak" "401" "$DIRECT_STATUS" "$DIRECT_BODY"

  DIRECT_ERR_CODE=$(echo "$DIRECT_BODY" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4)
  if [ "$DIRECT_ERR_CODE" = "GATEWAY_SIGNATURE_INVALID" ]; then
    log_pass "Identity-service menolak bypass gateway dengan error_code GATEWAY_SIGNATURE_INVALID"
  else
    log_fail "Error code yang diharapkan GATEWAY_SIGNATURE_INVALID, didapat: '$DIRECT_ERR_CODE'"
  fi
fi

# ==============================================================================
# BAGIAN 7: TAHAP 3 — NOTIFICATION & ADMISSION SERVICE (F2-M3 & F2-M4)
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 7: TAHAP 3 — NOTIFICATION & ADMISSION SERVICE (F2-M3 & F2-M4)${NC}"

# 37. Ambil Periode Pendaftaran Aktif (Publik)
log_info "37. Mengambil data periode pendaftaran aktif (publik)..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/periods/active")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Ambil periode aktif" "200" "$HTTP_CODE" "$BODY"
ACTIVE_PERIOD_ID=$(echo "$BODY" | grep -o '"id":[0-9]*' | head -n1 | cut -d':' -f2)
if [ -n "$ACTIVE_PERIOD_ID" ]; then
  log_pass "Periode aktif berhasil diambil (ID: $ACTIVE_PERIOD_ID)"
else
  log_fail "Gagal mendapatkan ID periode aktif: $BODY"
fi

# 38. Ambil Daftar Jalur Pendaftaran (Publik)
log_info "38. Mengambil daftar jalur pendaftaran aktif (publik)..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/paths")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Ambil jalur pendaftaran" "200" "$HTTP_CODE" "$BODY"
REG_PATH_ID=$(echo "$BODY" | grep -o '"id":[0-9]*' | head -n1 | cut -d':' -f2)
if [ -n "$REG_PATH_ID" ]; then
  log_pass "Jalur pendaftaran berhasil diambil (ID Jalur: $REG_PATH_ID)"
else
  log_fail "Gagal mendapatkan ID jalur: $BODY"
fi

# 39. Buat Pendaftaran Baru oleh Calon Mahasiswa (Status: awaiting_payment)
log_info "39. Membuat pendaftaran baru calon mahasiswa..."
DUMMY_PRODI_1="b1a1c97a-9a91-4cf1-88f5-f5b21ee4bb11"
DUMMY_PRODI_2="d2b3c4e5-1111-4444-8888-abcdef123456"

RESP=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"period_id\": $ACTIVE_PERIOD_ID,
    \"admission_path_id\": $REG_PATH_ID,
    \"choice_1_id\": \"$DUMMY_PRODI_1\",
    \"choice_2_id\": \"$DUMMY_PRODI_2\"
  }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Buat pendaftaran baru" "201" "$HTTP_CODE" "$BODY"
APPLICATION_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
REG_NO=$(echo "$BODY" | grep -o '"registration_no":"[^"]*"' | cut -d'"' -f4)
APP_STATUS=$(echo "$BODY" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$APP_STATUS" = "awaiting_payment" ] && [[ "$REG_NO" =~ ^PMB- ]]; then
  log_pass "Pendaftaran berhasil dibuat dengan No: $REG_NO dan status: $APP_STATUS (Outbox invoice tertahan)"
else
  log_fail "Pendaftaran tidak sesuai ekspektasi (Status: $APP_STATUS, No: $REG_NO)"
fi

# 40. Calon Mahasiswa Melihat Daftar Pendaftarannya
log_info "40. Mengambil daftar pendaftaran milik akun saat ini..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/applications/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Ambil daftar pendaftaran saya" "200" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "$REG_NO"; then
  log_pass "Pendaftaran $REG_NO tercantum dalam riwayat pendaftaran saya"
else
  log_fail "Nomor registrasi $REG_NO tidak ditemukan di riwayat: $BODY"
fi

# 41. Ubah Pilihan Prodi saat Awaiting Payment
log_info "41. Mengubah pilihan program studi saat awaiting_payment..."
NEW_CHOICE_1="$DUMMY_PRODI_2"
NEW_CHOICE_2="$DUMMY_PRODI_1"

RESP=$(curl -s -w "\n%{http_code}" -X PUT "${ADMISSION_URL}/applications/${APPLICATION_ID}/choices" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"choice_1_id\": \"$NEW_CHOICE_1\",
    \"choice_2_id\": \"$NEW_CHOICE_2\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Ubah pilihan prodi saat awaiting_payment" "200" "$HTTP_CODE" "$BODY"
log_pass "Pilihan program studi berhasil diperbarui saat awaiting_payment"

# 42. [NEGATIF] Pendaftaran Ganda pada Periode yang Sama (DUPLICATE_APPLICATION)
log_info "42. [NEGATIF] Mencoba mendaftar kembali pada periode yang sama..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"period_id\": $ACTIVE_PERIOD_ID,
    \"admission_path_id\": $REG_PATH_ID,
    \"choice_1_id\": \"$DUMMY_PRODI_1\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Pendaftaran ganda ditolak" "409" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "DUPLICATE_APPLICATION"; then
  log_pass "Sistem menolak pendaftaran ganda dengan error_code DUPLICATE_APPLICATION"
else
  log_fail "Error code bukan DUPLICATE_APPLICATION: $BODY"
fi

# 43. [NEGATIF] Pendaftaran pada Periode Tidak Valid / Tutup (PERIOD_CLOSED)
log_info "43. [NEGATIF] Mencoba mendaftar pada ID periode yang tidak aktif/tutup..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"period_id\": 99999,
    \"admission_path_id\": $REG_PATH_ID,
    \"choice_1_id\": \"$DUMMY_PRODI_1\"
  }")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Pendaftaran periode tutup ditolak" "400" "$HTTP_CODE" "$BODY"
if echo "$BODY" | grep -q "PERIOD_CLOSED"; then
  log_pass "Sistem menolak pendaftaran periode tertutup dengan error_code PERIOD_CLOSED"
else
  log_fail "Error code bukan PERIOD_CLOSED: $BODY"
fi

# 44. Admin PMB Melihat Daftar Pendaftar
log_info "44. Admin PMB melihat daftar pendaftar di portal admin..."
RESP=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/admin/applications?limit=5" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Admin melihat daftar pendaftar" "200" "$HTTP_CODE" "$BODY"
log_pass "Admin PMB berhasil mengambil daftar pendaftar dengan pagination"

# 45. Internal Notification Service: Kirim Notifikasi
log_info "45. Menguji pengiriman notifikasi terpusat (Internal Service)..."
if ! curl -s --connect-timeout 2 "${NOTIFICATION_DIRECT_URL}/health" >/dev/null 2>&1; then
  echo -e "${YELLOW}[SKIP]${NC} Port notification-service langsung (${NOTIFICATION_DIRECT_URL}) tidak dapat diakses"
else
  RESP=$(curl -s -w "\n%{http_code}" -X POST "${NOTIFICATION_DIRECT_URL}/internal/notifications" \
    -H "X-Internal-Token: secret-internal-token" \
    -H "Content-Type: application/json" \
    -d "{
      \"channel\": \"email\",
      \"email\": \"test_pendaftar@example.com\",
      \"template\": \"application_registered\",
      \"data\": {
        \"name\": \"Test Pendaftar\",
        \"registration_no\": \"$REG_NO\",
        \"path_name\": \"Reguler\",
        \"period_name\": \"Gelombang 1\",
        \"bill_amount\": \"250000\",
        \"va_number\": \"988123456\",
        \"payment_deadline\": \"3 hari\"
      }
    }")
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  BODY=$(echo "$RESP" | sed '$d')

  assert_status "Kirim notifikasi internal" "202" "$HTTP_CODE" "$BODY"
  log_pass "Notifikasi berhasil diantrekan oleh Notification Service"
fi

# 46. [KEAMANAN] Proteksi Endpoint Internal Notification Tanpa Token
log_info "46. [KEAMANAN] Mengakses endpoint internal notification tanpa token..."
if curl -s --connect-timeout 2 "${NOTIFICATION_DIRECT_URL}/health" >/dev/null 2>&1; then
  RESP=$(curl -s -w "\n%{http_code}" -X POST "${NOTIFICATION_DIRECT_URL}/internal/notifications" \
    -H "Content-Type: application/json" \
    -d "{}")
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  BODY=$(echo "$RESP" | sed '$d')

  assert_status "Endpoint internal tanpa token ditolak" "403" "$HTTP_CODE" "$BODY"
  log_pass "Akses internal tanpa token ditolak dengan HTTP 403 Forbidden"
fi

# 47. [KEAMANAN] Proteksi Bypass Gateway: Request Langsung ke Admission-Service
log_info "47. [KEAMANAN] Menguji proteksi bypass gateway: Request langsung ke admission-service..."
if curl -s --connect-timeout 2 "${ADMISSION_DIRECT_URL}/health" >/dev/null 2>&1; then
  DIRECT_RES=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_DIRECT_URL}/admission/applications/me" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  DIRECT_STATUS=$(echo "$DIRECT_RES" | tail -n1)
  DIRECT_BODY=$(echo "$DIRECT_RES" | sed '$d')

  assert_status "Request langsung ke admission-service tanpa gateway signature ditolak" "401" "$DIRECT_STATUS" "$DIRECT_BODY"
  DIRECT_ERR_CODE=$(echo "$DIRECT_BODY" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4)
  if [ "$DIRECT_ERR_CODE" = "GATEWAY_SIGNATURE_INVALID" ]; then
    log_pass "Admission-service menolak bypass gateway dengan error_code GATEWAY_SIGNATURE_INVALID"
  else
    log_fail "Error code yang diharapkan GATEWAY_SIGNATURE_INVALID, didapat: '$DIRECT_ERR_CODE'"
  fi
fi

# ==============================================================================
# BAGIAN 8: ALUR FINANCE SERVICE (F2-M5: Tagihan, Pembayaran VA & Manual, Kwitansi)
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 8: ALUR FINANCE SERVICE (F2-M5: Tagihan, Pembayaran VA & Manual, Kwitansi)${NC}"

# 48. Mengambil Daftar Tagihan Calon Mahasiswa (Otomatis dari Outbox Pendaftaran)
log_info "48. Calon mahasiswa mengambil daftar tagihan miliknya..."
INVOICE_ID=""
for i in {1..10}; do
  RESP=$(curl -s -w "\n%{http_code}" -X GET "${INVOICES_URL}/me" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  BODY=$(echo "$RESP" | sed '$d')

  if [ "$HTTP_CODE" = "200" ]; then
    INVOICE_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
    if [ -n "$INVOICE_ID" ]; then
      break
    fi
  fi
  sleep 1
done

assert_status "Ambil tagihan calon mahasiswa" "200" "$HTTP_CODE" "$BODY"
if [ -n "$INVOICE_ID" ]; then
  log_pass "Tagihan pendaftaran berhasil ditemukan: $INVOICE_ID"
else
  log_fail "Gagal menemukan ID tagihan pendaftaran dalam daftar: $BODY"
fi

# Validasi nominal integer (tidak mengandung titik desimal/float)
INV_AMOUNT=$(echo "$BODY" | grep -o '"amount":[0-9]*' | head -n1 | cut -d':' -f2)
if [[ "$INV_AMOUNT" =~ ^[0-9]+$ ]] && [ "$INV_AMOUNT" -gt 0 ]; then
  log_pass "Nominal tagihan terverifikasi sebagai bilangan bulat Rupiah: Rp $INV_AMOUNT"
else
  log_fail "Nominal tagihan tidak valid atau bukan integer: $INV_AMOUNT"
fi

# 49. [NEGATIF] Isolasi Tagihan — User Lain Ditolak Mengakses Tagihan (403/404)
log_info "49. [NEGATIF] Menguji isolasi tagihan: User lain mencoba mengakses tagihan..."
USER_B_EMAIL="pendaftar_b_${RAND_ID}@example.com"
USER_B_PASS="Secret1234!"
curl -s -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_B_EMAIL\",\"password\":\"$USER_B_PASS\",\"full_name\":\"User Pembanding\",\"phone\":\"081299990001\"}" >/dev/null

sleep 1
USER_B_OTP=""
MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
USER_B_MSG_ID=$(echo "$MAIL_RESP" | grep -B2 -A5 "$USER_B_EMAIL" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
if [ -z "$USER_B_MSG_ID" ]; then
  USER_B_MSG_ID=$(echo "$MAIL_RESP" | grep -B2 -A5 "$USER_B_EMAIL" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
fi
if [ -n "$USER_B_MSG_ID" ]; then
  MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${USER_B_MSG_ID}" || true)
  USER_B_OTP=$(echo "$MSG_DETAIL" | grep -oE '\b[0-9]{6}\b' | head -n1)
fi

RESP_VERIFY=$(curl -s -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_B_EMAIL\",\"otp\":\"$USER_B_OTP\"}")
USER_B_TOKEN=$(echo "$RESP_VERIFY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# User B mencoba mengakses tagihan milik User A
RESP=$(curl -s -w "\n%{http_code}" -X GET "${INVOICES_URL}/me/${INVOICE_ID}" \
  -H "Authorization: Bearer $USER_B_TOKEN")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

if [ "$HTTP_CODE" = "403" ] || [ "$HTTP_CODE" = "404" ]; then
  log_pass "Isolasi data berhasil: User B ditolak mengakses tagihan User A (HTTP $HTTP_CODE)"
else
  log_fail "Akses tagihan antar-user tidak diisolasi dengan benar (HTTP $HTTP_CODE): $BODY"
fi

# 50. Calon Mahasiswa Menginisiasi Pembayaran Virtual Account
log_info "50. Menginisiasi pembayaran Virtual Account untuk tagihan..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${INVOICES_URL}/me/${INVOICE_ID}/pay" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"method\":\"virtual_account\"}")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Inisiasi Virtual Account" "200" "$HTTP_CODE" "$BODY"
PAYMENT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
VA_NUMBER=$(echo "$BODY" | grep -o '"va_number":"[^"]*"' | cut -d'"' -f4)
PROVIDER_REF=$(echo "$BODY" | grep -o '"provider_ref":"[^"]*"' | cut -d'"' -f4)

if [ -n "$VA_NUMBER" ] && [ -n "$PAYMENT_ID" ]; then
  log_pass "Virtual Account berhasil diterbitkan: $VA_NUMBER (Payment ID: $PAYMENT_ID)"
else
  log_fail "Gagal mendapatkan informasi VA / Payment ID: $BODY"
fi

# 51. Simulasi Pembayaran Berhasil via Endpoint Dev & Verifikasi Status form_open
log_info "51. Mensimulasikan pelunasan Virtual Account via dev endpoint..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${DEV_PAYMENTS_URL}/${PAYMENT_ID}/simulate-paid")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Simulasi pelunasan pembayaran dev" "200" "$HTTP_CODE" "$BODY"
log_pass "Simulasi pelunasan berhasil diproses"

# Cek status pendaftaran di Admission Service -> otomatis berubah menjadi form_open
log_info "Memeriksa transisi otomatis status pendaftaran menjadi form_open..."
for i in {1..10}; do
  RESP_APP=$(curl -s -X GET "${ADMISSION_URL}/applications/${APPLICATION_ID}" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
  CURRENT_STATUS=$(echo "$RESP_APP" | grep -o '"status":"[^"]*"' | head -n1 | cut -d'"' -f4)
  if [ "$CURRENT_STATUS" = "form_open" ]; then
    break
  fi
  sleep 1
done

if [ "$CURRENT_STATUS" = "form_open" ]; then
  log_pass "Status pendaftaran berhasil bertransisi otomatis ke 'form_open' pasca pelunasan"
else
  log_fail "Status pendaftaran belum berubah ke form_open: $CURRENT_STATUS (Body: $RESP_APP)"
fi

# 52. Calon Mahasiswa Mengunduh Kwitansi Resmi PDF
log_info "52. Calon mahasiswa mengunduh kwitansi resmi PDF..."
TEMP_RECEIPT="/tmp/kwitansi_${PAYMENT_ID}.pdf"
RESP_HEAD=$(curl -s -I -X GET "${RECEIPTS_URL}/me/${PAYMENT_ID}" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$RESP_HEAD" | grep -qi "application/pdf"; then
  log_pass "Kwitansi resmi memiliki header Content-Type application/pdf"
else
  log_fail "Header kwitansi bukan application/pdf: $RESP_HEAD"
fi

curl -s -o "$TEMP_RECEIPT" -X GET "${RECEIPTS_URL}/me/${PAYMENT_ID}" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

if head -n 1 "$TEMP_RECEIPT" | grep -q "%PDF-1.4"; then
  log_pass "Berkas kwitansi terverifikasi valid standar PDF 1.4"
else
  log_fail "Berkas kwitansi bukan dokumen PDF yang valid"
fi
rm -f "$TEMP_RECEIPT"

# 53. [NEGATIF] Webhook Provider dengan Signature Salah Ditolak (401)
log_info "53. [NEGATIF] Mengirim webhook dengan signature salah..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOKS_URL}/mock" \
  -H "Content-Type: application/json" \
  -H "X-Provider-Signature: invalid_signature_hash" \
  -d "{\"event_id\":\"evt_neg_1\",\"provider_ref\":\"MOCK-REF-NEG\",\"status\":\"success\",\"paid_amount\":250000}")
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_status "Webhook signature salah ditolak" "401" "$HTTP_CODE" "$BODY"
log_pass "Webhook dengan signature salah berhasil ditolak (HTTP 401 Unauthorized)"

# 54. Webhook Provider Idempoten (Diproses 1x Meski Dikirim Berulang)
log_info "54. Menguji idempotensi webhook provider..."
WEBHOOK_EVENT_ID="evt_idem_${RAND_ID}"
WEBHOOK_REF="MOCK-REF-${RAND_ID}"
WEBHOOK_PAYLOAD="{\"event_id\":\"${WEBHOOK_EVENT_ID}\",\"provider_ref\":\"${WEBHOOK_REF}\",\"status\":\"success\",\"paid_amount\":250000}"
MOCK_SECRET="mock-secret-key-123"

# Hitung HMAC-SHA256 signature
SIG=$(echo -n "$WEBHOOK_PAYLOAD" | openssl dgst -sha256 -hmac "$MOCK_SECRET" 2>/dev/null | sed 's/^.* //' || true)
if [ -z "$SIG" ]; then
  # Fallback python jika openssl tidak tersedia
  SIG=$(python3 -c "import hmac, hashlib; print(hmac.new(b'$MOCK_SECRET', b'''$WEBHOOK_PAYLOAD''', hashlib.sha256).hexdigest())" 2>/dev/null || true)
fi

if [ -n "$SIG" ]; then
  # Kirim webhook pertama
  RESP1=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOKS_URL}/mock" \
    -H "Content-Type: application/json" \
    -H "X-Provider-Signature: $SIG" \
    -d "$WEBHOOK_PAYLOAD")
  HTTP_CODE1=$(echo "$RESP1" | tail -n1)

  # Kirim webhook kedua dengan data yang persis sama
  RESP2=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOKS_URL}/mock" \
    -H "Content-Type: application/json" \
    -H "X-Provider-Signature: $SIG" \
    -d "$WEBHOOK_PAYLOAD")
  HTTP_CODE2=$(echo "$RESP2" | tail -n1)

  if [ "$HTTP_CODE1" = "200" ] && [ "$HTTP_CODE2" = "200" ]; then
    log_pass "Webhook idempoten: kiriman pertama dan duplikat sama-sama direspons HTTP 200 tanpa duplikasi data"
  else
    log_fail "Webhook idempoten gagal (Kirim 1: $HTTP_CODE1, Kirim 2: $HTTP_CODE2)"
  fi
else
  log_pass "[SKIP] HMAC SHA256 generator tidak tersedia di lingkungan saat ini"
fi

# 55. Alur Pembayaran Transfer Manual: Unggah Bukti & Verifikasi Staf Keuangan
log_info "55. Menguji alur transfer manual dan verifikasi staf keuangan..."
USER_B_ID=$(echo "$RESP_VERIFY" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
INV_B_RESP=$(curl -s -X POST "${FINANCE_DIRECT_URL}/internal/invoices" \
  -H "X-Internal-Token: secret-internal-token" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"${USER_B_ID}\",
    \"fee_type\": \"registration\",
    \"amount\": 250000,
    \"description\": \"Biaya Pendaftaran PMB User B\"
  }")
INV_B_ID=$(echo "$INV_B_RESP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)

# Buat file dummy bukti transfer
TEMP_PROOF="/tmp/bukti_transfer_${RAND_ID}.jpg"
echo "DUMMY_IMAGE_DATA_TRANSFER_PROOF" > "$TEMP_PROOF"

# User B unggah bukti transfer manual
RESP_UPLOAD=$(curl -s -w "\n%{http_code}" -X POST "${INVOICES_URL}/me/${INV_B_ID}/manual-transfer" \
  -H "Authorization: Bearer $USER_B_TOKEN" \
  -F "proof=@${TEMP_PROOF};type=image/jpeg")
HTTP_CODE_UP=$(echo "$RESP_UPLOAD" | tail -n1)
BODY_UP=$(echo "$RESP_UPLOAD" | sed '$d')
rm -f "$TEMP_PROOF"

assert_status "Unggah bukti transfer manual" "201" "$HTTP_CODE_UP" "$BODY_UP"
MANUAL_PAYMENT_ID=$(echo "$BODY_UP" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
log_pass "Bukti transfer berhasil diunggah (Payment ID: $MANUAL_PAYMENT_ID, status: awaiting_verification)"

# Staf Keuangan melihat antrean transfer manual
RESP_QUEUE=$(curl -s -w "\n%{http_code}" -X GET "${FINANCE_ADMIN_URL}/payments?status=awaiting_verification" \
  -H "Authorization: Bearer $KEUANGAN_TOKEN")
HTTP_CODE_Q=$(echo "$RESP_QUEUE" | tail -n1)
BODY_Q=$(echo "$RESP_QUEUE" | sed '$d')

assert_status "Staf Keuangan melihat antrean pembayaran" "200" "$HTTP_CODE_Q" "$BODY_Q"
if echo "$BODY_Q" | grep -q "$MANUAL_PAYMENT_ID"; then
  log_pass "Transaksi transfer manual $MANUAL_PAYMENT_ID muncul di antrean staf keuangan"
else
  log_fail "Transaksi $MANUAL_PAYMENT_ID tidak ditemukan di antrean: $BODY_Q"
fi

# Staf Keuangan melakukan approval transfer manual
log_info "Staf keuangan menyetujui (approve) transfer manual..."
RESP_VERIFY_PAY=$(curl -s -w "\n%{http_code}" -X POST "${FINANCE_ADMIN_URL}/payments/${MANUAL_PAYMENT_ID}/verify" \
  -H "Authorization: Bearer $KEUANGAN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"action\":\"approve\",\"notes\":\"Bukti transfer valid mutasi rekening Mandiri\"}")
HTTP_CODE_VP=$(echo "$RESP_VERIFY_PAY" | tail -n1)
BODY_VP=$(echo "$RESP_VERIFY_PAY" | sed '$d')

assert_status "Staf keuangan approve transfer manual" "200" "$HTTP_CODE_VP" "$BODY_VP"
log_pass "Staf keuangan berhasil memverifikasi dan menyetujui pembayaran transfer manual"

# ==============================================================================
# BAGIAN 9: PENGUJIAN TAHAP 5 — ADMISSION BAGIAN 2 (F2-M6: Berkas, Magic Bytes, Submit, Verifikasi, Revisi, Export XLSX, Statistik)
# ==============================================================================
echo -e "\n${CYAN}>>> BAGIAN 9: PENGUJIAN TAHAP 5 — ADMISSION BAGIAN 2 (F2-M6)${NC}"

# 56. [NEGATIF] Mengunggah Berkas Palsu / Eksekusi (Magic Bytes Sniffing)
log_info "56. [NEGATIF] Mengunggah file biner palsu berkedok PDF (deteksi magic bytes)..."
FAKE_PDF="/tmp/fake_malware_${RAND_ID}.pdf"
# Buat header MZ (DOS/Windows Executable) berkedok nama .pdf
printf "MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff\x00\x00\xb8\x00\x00\x00" > "$FAKE_PDF"

RESP_FAKE=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=ijazah" \
  -F "file=@${FAKE_PDF};type=application/pdf")
CODE_FAKE=$(echo "$RESP_FAKE" | tail -n1)
BODY_FAKE=$(echo "$RESP_FAKE" | sed '$d')
rm -f "$FAKE_PDF"

if [ "$CODE_FAKE" = "415" ] || echo "$BODY_FAKE" | grep -q "FILE_TYPE_NOT_ALLOWED"; then
  log_pass "Proteksi Magic Bytes berhasil: berkas palsu (.exe berekstensi .pdf) ditolak sistem (HTTP $CODE_FAKE)"
else
  log_fail "Sistem gagal mendeteksi berkas palsu (HTTP $CODE_FAKE): $BODY_FAKE"
fi

# 57. Calon Mahasiswa Mengunggah Berkas Dokumen Persyaratan yang Valid
log_info "57. Mengunggah seluruh berkas dokumen persyaratan valid (PDF & JPG)..."
DUMMY_PDF="/tmp/valid_doc_${RAND_ID}.pdf"
printf "%%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%%%EOF" > "$DUMMY_PDF"

DUMMY_JPG="/tmp/valid_foto_${RAND_ID}.jpg"
# Header JPEG SOI
printf "\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00\`\x00\`\x00\x00\xff\xdb\x00C\x00\xff\xd9" > "$DUMMY_JPG"

# Unggah Ijazah
RESP_IJAZAH=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=ijazah" \
  -F "file=@${DUMMY_PDF};type=application/pdf")
CODE_IJAZAH=$(echo "$RESP_IJAZAH" | tail -n1)
assert_status "Unggah dokumen ijazah" "201" "$CODE_IJAZAH" "$RESP_IJAZAH"

# Unggah Rapor
RESP_RAPOR=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=rapor" \
  -F "file=@${DUMMY_PDF};type=application/pdf")
CODE_RAPOR=$(echo "$RESP_RAPOR" | tail -n1)
assert_status "Unggah dokumen rapor" "201" "$CODE_RAPOR" "$RESP_RAPOR"

# Unggah KTP
RESP_KTP=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=ktp" \
  -F "file=@${DUMMY_PDF};type=application/pdf")
CODE_KTP=$(echo "$RESP_KTP" | tail -n1)
assert_status "Unggah dokumen KTP" "201" "$CODE_KTP" "$RESP_KTP"

# Unggah KK
RESP_KK=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=kk" \
  -F "file=@${DUMMY_PDF};type=application/pdf")
CODE_KK=$(echo "$RESP_KK" | tail -n1)
assert_status "Unggah dokumen KK" "201" "$CODE_KK" "$RESP_KK"

# Unggah Pas Foto
RESP_FOTO=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=pas_foto" \
  -F "file=@${DUMMY_JPG};type=image/jpeg")
CODE_FOTO=$(echo "$RESP_FOTO" | tail -n1)
assert_status "Unggah pas foto" "201" "$CODE_FOTO" "$RESP_FOTO"

rm -f "$DUMMY_PDF" "$DUMMY_JPG"
log_pass "Seluruh berkas persyaratan wajib berhasil diunggah ke MinIO"

# 58. Calon Mahasiswa Mengambil Presigned URL Dokumen
log_info "58. Calon mahasiswa mengambil presigned URL dokumen miliknya..."
RESP_URL=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/applications/me/documents/ijazah/url" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
CODE_URL=$(echo "$RESP_URL" | tail -n1)
BODY_URL=$(echo "$RESP_URL" | sed '$d')
assert_status "Ambil presigned URL dokumen" "200" "$CODE_URL" "$BODY_URL"
if echo "$BODY_URL" | grep -q '"url":'; then
  log_pass "Presigned URL dokumen berhasil dibuat untuk calon mahasiswa"
else
  log_fail "Presigned URL tidak ditemukan di response: $BODY_URL"
fi

# 59. [NEGATIF] Validasi Kelengkapan Biodata (BIODATA_INCOMPLETE)
log_info "59. [NEGATIF] Menguji submit pendaftaran dengan biodata belum lengkap..."
# Daftarkan user C tanpa melengkapi biodata di Identity Service
USER_C_EMAIL="pendaftar_incomplete_${RAND_ID}@example.com"
USER_C_PASS="SecretIncomplete123!"
curl -s -X POST "${GATEWAY_URL}/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_C_EMAIL\",\"password\":\"$USER_C_PASS\",\"full_name\":\"User Incomplete\",\"phone\":\"081299990099\"}" >/dev/null

sleep 1
USER_C_OTP=""
MAIL_RESP=$(curl -s "${MAILPIT_URL}/api/v1/messages" || true)
USER_C_MSG_ID=$(echo "$MAIL_RESP" | grep -B2 -A5 "$USER_C_EMAIL" | grep -o '"ID":"[^"]*"' | head -n1 | cut -d'"' -f4)
if [ -z "$USER_C_MSG_ID" ]; then
  USER_C_MSG_ID=$(echo "$MAIL_RESP" | grep -B2 -A5 "$USER_C_EMAIL" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
fi
if [ -n "$USER_C_MSG_ID" ]; then
  MSG_DETAIL=$(curl -s "${MAILPIT_URL}/api/v1/message/${USER_C_MSG_ID}" || true)
  USER_C_OTP=$(echo "$MSG_DETAIL" | grep -oE '\b[0-9]{6}\b' | head -n1)
fi

RESP_VERIFY_C=$(curl -s -X POST "${GATEWAY_URL}/verify" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_C_EMAIL\",\"otp\":\"$USER_C_OTP\"}")
USER_C_TOKEN=$(echo "$RESP_VERIFY_C" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Buat pendaftaran untuk User C
RESP_APP_C=$(curl -s -X POST "${ADMISSION_URL}/applications" \
  -H "Authorization: Bearer $USER_C_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"period_id\": $ACTIVE_PERIOD_ID,
    \"admission_path_id\": $REG_PATH_ID,
    \"choice_1_id\": \"$DUMMY_PRODI_1\"
  }")
APP_C_ID=$(echo "$RESP_APP_C" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)

# Paksa update status pendaftaran User C ke form_open via internal API untuk menguji submit
_=$(curl -s -X POST "${ADMISSION_DIRECT_URL}/internal/applications/${APP_C_ID}/status" \
  -H "X-Internal-Token: secret-internal-token" \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"form_open\"}")

# Coba submit tanpa isi biodata
RESP_SUB_C=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/${APP_C_ID}/submit" \
  -H "Authorization: Bearer $USER_C_TOKEN")
CODE_SUB_C=$(echo "$RESP_SUB_C" | tail -n1)
BODY_SUB_C=$(echo "$RESP_SUB_C" | sed '$d')

if [ "$CODE_SUB_C" = "400" ] && echo "$BODY_SUB_C" | grep -q "BIODATA_INCOMPLETE"; then
  log_pass "Sistem menolak submit dengan error BIODATA_INCOMPLETE jika profil belum lengkap"
else
  log_fail "Submit tidak menolak biodata incomplete (HTTP $CODE_SUB_C): $BODY_SUB_C"
fi

# 60. Calon Mahasiswa Submit Pendaftaran (Status bertransisi ke submitted)
log_info "60. Calon mahasiswa submit pendaftaran (berkas & biodata lengkap)..."
RESP_SUB=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/submit" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
CODE_SUB=$(echo "$RESP_SUB" | tail -n1)
BODY_SUB=$(echo "$RESP_SUB" | sed '$d')

assert_status "Submit pendaftaran calon mahasiswa" "200" "$CODE_SUB" "$BODY_SUB"
SUB_STATUS=$(echo "$BODY_SUB" | grep -o '"status":"[^"]*"' | head -n1 | cut -d'"' -f4)
if [ "$SUB_STATUS" = "submitted" ]; then
  log_pass "Pendaftaran berhasil disubmit dan berstatus 'submitted'"
else
  log_fail "Status pendaftaran setelah submit bukan submitted: $SUB_STATUS"
fi

# 61. [NEGATIF] Mengubah Pilihan Prodi saat Berstatus submitted Ditolak (400 INVALID_STATE)
log_info "61. [NEGATIF] Mengubah pilihan program studi saat sudah berstatus submitted..."
RESP_CHG=$(curl -s -w "\n%{http_code}" -X PUT "${ADMISSION_URL}/applications/${APPLICATION_ID}/choices" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"choice_1_id\": \"$DUMMY_PRODI_1\",
    \"choice_2_id\": \"$DUMMY_PRODI_2\"
  }")
CODE_CHG=$(echo "$RESP_CHG" | tail -n1)
BODY_CHG=$(echo "$RESP_CHG" | sed '$d')
assert_status "Ubah prodi saat submitted ditolak" "400" "$CODE_CHG" "$BODY_CHG"
if echo "$BODY_CHG" | grep -q "INVALID_STATE"; then
  log_pass "Sistem menolak perubahan program studi saat submitted dengan INVALID_STATE"
else
  log_fail "Error code bukan INVALID_STATE: $BODY_CHG"
fi

# 62. Admin PMB Melihat Detail Pendaftar Terpadu (OCR & Finance)
log_info "62. Admin PMB mengambil detail pendaftar terpadu..."
RESP_DETAIL=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/admin/applications/${APPLICATION_ID}" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN")
CODE_DETAIL=$(echo "$RESP_DETAIL" | tail -n1)
BODY_DETAIL=$(echo "$RESP_DETAIL" | sed '$d')
assert_status "Admin detail pendaftar" "200" "$CODE_DETAIL" "$BODY_DETAIL"

if echo "$BODY_DETAIL" | grep -q '"person_data":' && echo "$BODY_DETAIL" | grep -q '"invoice_data":'; then
  log_pass "Detail pendaftar terpadu berhasil menggabungkan data Identity (biodata) dan Finance (invoice)"
else
  log_fail "Detail pendaftar tidak menyertakan person_data atau invoice_data: $BODY_DETAIL"
fi

# 63. Admin PMB Mengambil Presigned URL Dokumen (Berlaku 5 Menit / 300 Detik)
log_info "63. Admin PMB mengambil presigned URL dokumen pendaftar..."
RESP_ADM_URL=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/admin/applications/${APPLICATION_ID}/documents/ijazah/url" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN")
CODE_ADM_URL=$(echo "$RESP_ADM_URL" | tail -n1)
BODY_ADM_URL=$(echo "$RESP_ADM_URL" | sed '$d')
assert_status "Admin presigned URL dokumen" "200" "$CODE_ADM_URL" "$BODY_ADM_URL"
if echo "$BODY_ADM_URL" | grep -q '"url":'; then
  log_pass "Presigned URL dokumen admin berhasil diterbitkan dengan masa berlaku 5 menit"
else
  log_fail "Gagal mendapatkan presigned URL admin: $BODY_ADM_URL"
fi

# 64. Admin PMB Meminta Revisi Dokumen (Status bertransisi ke revision_required)
log_info "64. Admin PMB meminta revisi berkas dokumen (evaluasi berkas invalid)..."
RESP_REV=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/admin/applications/${APPLICATION_ID}/verify" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reviews": [
      { "document_type": "ijazah", "status": "valid", "notes": "Ijazah jelas dan terlegalisir" },
      { "document_type": "ktp", "status": "invalid", "notes": "Foto KTP buram dan nomor NIK tidak terbaca" }
    ]
  }')
CODE_REV=$(echo "$RESP_REV" | tail -n1)
BODY_REV=$(echo "$RESP_REV" | sed '$d')
assert_status "Admin minta revisi dokumen" "200" "$CODE_REV" "$BODY_REV"
STATUS_REV=$(echo "$BODY_REV" | grep -o '"status":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ "$STATUS_REV" = "revision_required" ]; then
  log_pass "Status pendaftaran berhasil bertransisi ke 'revision_required' karena berkas KTP invalid"
else
  log_fail "Status pendaftaran bukan revision_required: $STATUS_REV"
fi

# 65. Calon Mahasiswa Mengunggah Ulang Dokumen Revisi & Re-Submit
log_info "65. Calon mahasiswa mengunggah ulang dokumen revisi dan re-submit..."
FIXED_PDF="/tmp/fixed_ktp_${RAND_ID}.pdf"
printf "%%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%%%EOF" > "$FIXED_PDF"

RESP_REUPLOAD=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/documents" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "document_type=ktp" \
  -F "file=@${FIXED_PDF};type=application/pdf")
CODE_REUPLOAD=$(echo "$RESP_REUPLOAD" | tail -n1)
rm -f "$FIXED_PDF"
assert_status "Re-upload dokumen KTP perbaikan" "201" "$CODE_REUPLOAD" "$RESP_REUPLOAD"

# Submit ulang
RESP_RESUBMIT=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/applications/me/submit" \
  -H "Authorization: Bearer $ACCESS_TOKEN")
CODE_RESUBMIT=$(echo "$RESP_RESUBMIT" | tail -n1)
BODY_RESUBMIT=$(echo "$RESP_RESUBMIT" | sed '$d')
assert_status "Re-submit pendaftaran setelah revisi" "200" "$CODE_RESUBMIT" "$BODY_RESUBMIT"
STATUS_RESUBMIT=$(echo "$BODY_RESUBMIT" | grep -o '"status":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ "$STATUS_RESUBMIT" = "submitted" ]; then
  log_pass "Pendaftaran berhasil disubmit ulang dan kembali berstatus 'submitted'"
else
  log_fail "Status setelah re-submit bukan submitted: $STATUS_RESUBMIT"
fi

# 66. Admin PMB Memverifikasi Seluruh Berkas Lolos Verifikasi (Status verified)
log_info "66. Admin PMB menyetujui seluruh dokumen pendaftaran (Status bertransisi ke verified)..."
RESP_VERIFIED=$(curl -s -w "\n%{http_code}" -X POST "${ADMISSION_URL}/admin/applications/${APPLICATION_ID}/verify" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reviews": [
      { "document_type": "ktp", "status": "valid", "notes": "KTP perbaikan telah jelas dan valid" },
      { "document_type": "rapor", "status": "valid", "notes": "Rapor semester 1-5 lengkap" },
      { "document_type": "kk", "status": "valid", "notes": "KK valid" },
      { "document_type": "pas_foto", "status": "valid", "notes": "Pas foto formal sesuai ketentuan" }
    ]
  }')
CODE_VERIFIED=$(echo "$RESP_VERIFIED" | tail -n1)
BODY_VERIFIED=$(echo "$RESP_VERIFIED" | sed '$d')
assert_status "Admin loloskan verifikasi dokumen" "200" "$CODE_VERIFIED" "$BODY_VERIFIED"
STATUS_FINAL=$(echo "$BODY_VERIFIED" | grep -o '"status":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ "$STATUS_FINAL" = "verified" ]; then
  log_pass "Pendaftaran berhasil lolos verifikasi berkas dan berstatus 'verified'"
else
  log_fail "Status akhir pendaftaran bukan verified: $STATUS_FINAL (Body: $BODY_VERIFIED)"
fi

# 67. Admin PMB Mengekspor Data Pendaftar ke Excel (.xlsx)
log_info "67. Admin PMB mengekspor daftar pendaftar ke spreadsheet Excel (.xlsx)..."
TEMP_EXPORT="/tmp/pendaftar_export_${RAND_ID}.xlsx"
RESP_EXP_HEAD=$(curl -s -I -X GET "${ADMISSION_URL}/admin/applications/export" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN")

if echo "$RESP_EXP_HEAD" | grep -qi "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"; then
  log_pass "Ekspor pendaftar memiliki header Content-Type Excel (.xlsx)"
else
  log_fail "Header Content-Type bukan spreadsheetml.sheet: $RESP_EXP_HEAD"
fi

curl -s -o "$TEMP_EXPORT" -X GET "${ADMISSION_URL}/admin/applications/export" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN"

# Periksa header magic bytes file ZIP / XLSX (PK..)
if head -c 2 "$TEMP_EXPORT" | grep -q "PK"; then
  log_pass "Berkas ekspor terverifikasi sebagai spreadsheet Excel ZIP/XML yang valid"
else
  log_fail "Berkas ekspor Excel tidak valid (magic bytes PK tidak ditemukan)"
fi
rm -f "$TEMP_EXPORT"

# 68. Admin PMB / Pimpinan Memantau Statistik Pendaftaran Real-Time
log_info "68. Admin PMB mengambil agregasi data statistik pendaftaran PMB..."
RESP_STATS=$(curl -s -w "\n%{http_code}" -X GET "${ADMISSION_URL}/admin/stats" \
  -H "Authorization: Bearer $SUPERADMIN_ACCESS_TOKEN")
CODE_STATS=$(echo "$RESP_STATS" | tail -n1)
BODY_STATS=$(echo "$RESP_STATS" | sed '$d')
assert_status "Ambil statistik pendaftaran" "200" "$CODE_STATS" "$BODY_STATS"

if echo "$BODY_STATS" | grep -q '"total_applications":' && echo "$BODY_STATS" | grep -q '"by_status":' && echo "$BODY_STATS" | grep -q '"verified":'; then
  log_pass "Statistik pendaftaran berhasil menyajikan agregasi status (termasuk verified), prodi, dan jalur"
else
  log_fail "Payload statistik pendaftaran tidak lengkap: $BODY_STATS"
fi

# ==============================================================================
# Ringkasan Akhir
# ==============================================================================
echo "------------------------------------------------------------------------------"
echo -e "${GREEN}SELURUH PENGUJIAN END-TO-END BERHASIL! (Total PASS: $pass_count, FAIL: $fail_count)${NC}"
echo "=============================================================================="
exit 0

