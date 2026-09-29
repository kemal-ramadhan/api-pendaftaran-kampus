# Duitku Sandbox Verification Script
# Requirement: PowerShell 5.1+, ASCII-only encoding

param (
    [string]$GatewayUrl = "http://localhost:8000",
    [string]$MailpitUrl = "http://localhost:8025",
    [switch]$TestReconcile,
    [string]$OutputFile = "duitku-sandbox-report.txt"
)

$ErrorActionPreference = "Stop"

function Write-Log {
    param ([string]$Message, [string]$Level = "INFO")
    $timestamp = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
    Write-Host "[$timestamp] [$Level] $Message"
}

function Mask-String {
    param ([string]$Str, [int]$Keep = 4)
    if ([string]::IsNullOrEmpty($Str)) { return "N/A" }
    if ($Str.Length -le $Keep) { return "***" }
    return $Str.Substring(0, $Keep) + ("*" * ($Str.Length - $Keep))
}

Write-Log "=========================================================="
Write-Log "MEMULAI PENGUJIAN INTEGRASI DUITKU SANDBOX (CR-008)"
Write-Log "=========================================================="

# 1. Cek Konfigurasi Lingkungan
Write-Log "Memeriksa konfigurasi backend..."
try {
    $healthResp = Invoke-RestMethod -Uri "$GatewayUrl/health" -Method Get -TimeoutSec 5
    Write-Log "API Gateway sehat (Status: $($healthResp.status))"
} catch {
    Write-Log "Gagal menghubungi API Gateway di $GatewayUrl. Pastikan seluruh service berjalan." "ERROR"
    exit 1
}

# 2. Pendaftaran Akun Mahasiswa Baru untuk Pengujian
$randomId = (Get-Random -Minimum 1000 -Maximum 9999)
$testEmail = "duitku.tester.$randomId@pmb.local"
$testPassword = "PasswordTester123!"
$testName = "Tester Duitku $randomId"

Write-Log "Mendaftarkan calon mahasiswa uji: $testEmail"
$registerBody = @{
    email = $testEmail
    password = $testPassword
    full_name = $testName
    phone_number = "08123456$randomId"
} | ConvertTo-Json

try {
    $regResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/auth/register" -Method Post -Body $registerBody -ContentType "application/json"
    Write-Log "Pendaftaran berhasil, menunggu OTP..."
} catch {
    Write-Log "Gagal mendaftarkan user baru: $_" "ERROR"
    exit 1
}

Start-Sleep -Seconds 2

# 3. Pengambilan OTP dari Mailpit
$otp = "123456" # Fallback dev OTP
try {
    $mailResp = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/messages" -Method Get -TimeoutSec 5
    if ($mailResp.messages -and $mailResp.messages.Count -gt 0) {
        $latestMsg = $mailResp.messages | Where-Object { $_.To[0].Address -eq $testEmail } | Select-Object -First 1
        if ($latestMsg) {
            $snippet = $latestMsg.Snippet
            if ($snippet -match "\b(\d{6})\b") {
                $otp = $matches[1]
                Write-Log "OTP berhasil diambil dari Mailpit: $otp"
            }
        }
    }
} catch {
    Write-Log "Gagal membaca Mailpit, mencoba verifikasi langsung dengan OTP dev..." "WARN"
}

# 4. Verifikasi Akun & Login
$verifyBody = @{
    email = $testEmail
    otp = $otp
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri "$GatewayUrl/api/v1/auth/verify" -Method Post -Body $verifyBody -ContentType "application/json" | Out-Null
    Write-Log "Verifikasi akun berhasil"
} catch {
    Write-Log "Verifikasi akun gagal: $_" "WARN"
}

$loginBody = @{
    email = $testEmail
    password = $testPassword
} | ConvertTo-Json

$loginResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $loginResp.data.access_token
if (-not $token) {
    Write-Log "Gagal memperoleh access token login" "ERROR"
    exit 1
}
Write-Log "Login berhasil, token JWT diperoleh"

$headers = @{
    "Authorization" = "Bearer $token"
}

# 5. Cek Daftar Metode Pembayaran Sandbox
Write-Log "Mengambil daftar metode pembayaran dari Duitku WebAPI..."
try {
    $methodsResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/finance/payment-methods?amount=250000" -Method Get -Headers $headers
    $methods = $methodsResp.data
    Write-Log "Berhasil mengambil $($methods.Count) metode pembayaran Duitku:"
    foreach ($m in $methods) {
        Write-Log " - [$($m.paymentMethod)] $($m.paymentName) (Fee: Rp $($m.totalFee))"
    }
} catch {
    Write-Log "Gagal mengambil metode pembayaran dari gateway: $_" "WARN"
}

# 6. Buat Formulir Pendaftaran Baru
Write-Log "Membuat formulir pendaftaran baru..."
$activePeriods = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/admission/periods/active" -Method Get
$periodId = $activePeriods.data.id
$paths = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/admission/paths" -Method Get
$pathId = $paths.data[0].id

$appBody = @{
    period_id = $periodId
    admission_path_id = $pathId
    choice_1_id = "00000000-0000-0000-0000-000000000001"
} | ConvertTo-Json

try {
    $appResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/admission/applications" -Method Post -Headers $headers -Body $appBody -ContentType "application/json"
    $appId = $appResp.data.id
    Write-Log "Formulir pendaftaran berhasil dibuat: ID $appId (Status: $($appResp.data.status))"
} catch {
    Write-Log "Gagal membuat formulir pendaftaran: $_" "ERROR"
    exit 1
}

# 7. Ambil Tagihan Pendaftaran
Write-Log "Mengambil informasi tagihan pendaftaran..."
$invoicesResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/invoices/me" -Method Get -Headers $headers
$invoice = $invoicesResp.data.items[0]
if (-not $invoice) {
    Write-Log "Tagihan pendaftaran tidak ditemukan" "ERROR"
    exit 1
}
$invoiceId = $invoice.id
$amount = $invoice.amount
Write-Log "Tagihan pendaftaran ditemukan: $invoiceId, Jumlah: Rp $amount (Status: $($invoice.status))"

# 8. Request Pembayaran Virtual Account
Write-Log "Melakukan checkout tagihan dengan Virtual Account Duitku..."
$payBody = @{
    method = "virtual_account"
} | ConvertTo-Json

try {
    $payResp = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/invoices/me/$invoiceId/pay" -Method Post -Headers $headers -Body $payBody -ContentType "application/json"
    $paymentData = $payResp.data
    $vaNumber = $paymentData.va_number
    $paymentUrl = $paymentData.payment_url
    $paymentId = $paymentData.id

    Write-Log "Virtual Account Duitku Berhasil Dibuat!"
    Write-Log "----------------------------------------------------------"
    Write-Log "Nomor Virtual Account : $vaNumber"
    Write-Log "Jumlah Pembayaran     : Rp $amount"
    Write-Log "URL Pembayaran POP    : $paymentUrl"
    Write-Log "Payment ID Sistem     : $paymentId"
    Write-Log "----------------------------------------------------------"
} catch {
    Write-Log "Gagal inisialisasi pembayaran ke Duitku: $_" "ERROR"
    exit 1
}

# 9. Polling Status Tagihan Menunggu Pembayaran
Write-Log "Menunggu konfirmasi pembayaran dari Duitku..."
Write-Log "Silakan lakukan simulasi bayar pada simulator Duitku sandbox atau gunakan payment_url di atas."
Write-Log "Memulai polling status setiap 10 detik (maksimal 10 menit)..."

$maxAttempts = 60
$attempt = 0
$isPaid = $false
$resolutionType = "unknown"

while ($attempt -lt $maxAttempts) {
    $attempt++
    Start-Sleep -Seconds 10

    try {
        $invCheck = Invoke-RestMethod -Uri "$GatewayUrl/api/v1/invoices/me/$invoiceId" -Method Get -Headers $headers
        $invStatus = $invCheck.data.status
        $paidAmount = $invCheck.data.paid_amount

        Write-Log "Polling #$attempt - Status Tagihan: $invStatus (Terbayar: Rp $paidAmount)"

        if ($invStatus -eq "paid") {
            $isPaid = $true
            $resolutionType = "webhook_or_reconcile"
            break
        }
    } catch {
        Write-Log "Peringatan polling status: $_" "WARN"
    }
}

# 10. Ringkasan & Pelaporan Hasil
$reportLines = @()
$reportLines += "=========================================================="
$reportLines += "LAPORAN HASIL PENGUJIAN INTEGRASI DUITKU SANDBOX (CR-008)"
$reportLines += "Waktu Pengujian : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
$reportLines += "Email Akun Uji  : $testEmail"
$reportLines += "Invoice ID      : $invoiceId"
$reportLines += "Nomor VA Duitku : $vaNumber"
$reportLines += "Jumlah Tagihan  : Rp $amount"

if ($isPaid) {
    Write-Log "=========================================================="
    Write-Log "HASIL PENGUJIAN: PASS - Pembayaran Berhasil Dikonfirmasi!"
    Write-Log "Status Tagihan: PAID"
    Write-Log "=========================================================="
    $reportLines += "Hasil Pengujian : PASS (Lunas terkonfirmasi)"
} else {
    Write-Log "==========================================================" "WARN"
    Write-Log "HASIL PENGUJIAN: PENDING / TIMEOUT" "WARN"
    Write-Log "Pembayaran belum terdeteksi lunas dalam jendela waktu polling." "WARN"
    Write-Log "Pastikan simulator Duitku telah diproses atau webhook tunnel aktif." "WARN"
    Write-Log "==========================================================" "WARN"
    $reportLines += "Hasil Pengujian : PENDING / TIMEOUT (Belum terkonfirmasi pada waktu uji)"
}

$reportLines += "=========================================================="
$reportLines | Out-File -FilePath $OutputFile -Encoding ascii

Write-Log "Laporan pengujian disimpan ke: $OutputFile"
Write-Log "Pengujian Duitku sandbox check selesai."
