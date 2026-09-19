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

# ==============================================================================
# Ringkasan Akhir
# ==============================================================================
echo "------------------------------------------------------------------------------"
echo -e "${GREEN}SELURUH PENGUJIAN END-TO-END BERHASIL! (Total PASS: $pass_count, FAIL: $fail_count)${NC}"
echo "=============================================================================="
exit 0
