<#
  final-check.ps1 - Pengecekan akhir backend PMB (tanpa cek manual)
  Jalankan dari root repo, pilih tahap:
      powershell -ExecutionPolicy Bypass -File scripts/final-check.ps1 -Tahap rilis        (fase R: CR-013)
      powershell -ExecutionPolicy Bypass -File scripts/final-check.ps1 -Tahap konsolidasi  (fase K: + CR-011, CR-012, 3 service)
      powershell -ExecutionPolicy Bypass -File scripts/final-check.ps1 -Tahap penuh        (fase F: + CR-014 referal)
  Semua tahap dijalankan di branch main.
  Hasil: ringkasan di layar + file final-check-report.txt (kirim file ini ke Claude).
#>
param([string]$Tahap = "")

if (@("rilis","konsolidasi","penuh") -notcontains $Tahap) {
    Write-Host "Parameter -Tahap wajib diisi: rilis | konsolidasi | penuh" -ForegroundColor Yellow
    Write-Host "Contoh: powershell -ExecutionPolicy Bypass -File scripts\final-check.ps1 -Tahap rilis"
    exit 2
}

$ErrorActionPreference = "Continue"
$Root    = (Get-Location).Path
$Compose = "deploy/docker-compose.yml"
$Report  = Join-Path $Root "final-check-report.txt"
$GitBash = "C:\Program Files\Git\bin\bash.exe"
$GoImage = "golang:1.26-alpine"
$Results = New-Object System.Collections.ArrayList
$Log     = New-Object System.Collections.ArrayList

function Add-Result($name, $ok, $detail) {
    [void]$Results.Add([pscustomobject]@{ Cek = $name; Status = $(if ($ok) { "OK" } else { "GAGAL" }); Detail = $detail })
    $color = if ($ok) { "Green" } else { "Red" }
    Write-Host ("[{0}] {1} - {2}" -f $(if ($ok) { "OK" } else { "GAGAL" }), $name, $detail) -ForegroundColor $color
}
function Log($text) { [void]$Log.Add($text) }
function Section($t) { Write-Host "`n=== $t ===" -ForegroundColor Cyan; Log "`n=== $t ===" }

# ---------------------------------------------------------------- 0. Prasyarat
Section "0. Prasyarat"
docker info *> $null
$dockerOk = ($LASTEXITCODE -eq 0)
Add-Result "Docker Desktop menyala" $dockerOk $(if ($dockerOk) { "docker info OK" } else { "Nyalakan Docker Desktop lalu ulangi" })
if (-not $dockerOk) { $Results | Format-Table -AutoSize; exit 1 }
$gitBashOk = Test-Path $GitBash
Add-Result "Git Bash tersedia" $gitBashOk $GitBash
$goLocal = [bool](Get-Command go -ErrorAction SilentlyContinue)

# ---------------------------------------------------------------- 1. Git sinkron
Section "1. Sinkronisasi Git (tahap: $Tahap)"
$Branch = (git rev-parse --abbrev-ref HEAD).Trim()
$branchOk = ($Branch -eq "main")
Add-Result "Branch sesuai tahap" $branchOk $(if ($branchOk) { "branch $Branch" } else { "Tahap $Tahap harus di branch main (sekarang $Branch). Jalankan: git checkout main" })
git fetch origin *> $null
git pull --rebase origin $Branch 2>&1 | ForEach-Object { Log $_ }
$dirty = git status --porcelain
Add-Result "Working tree bersih" ([string]::IsNullOrWhiteSpace(($dirty -join ""))) $(if ($dirty) { "Ada perubahan belum di-commit: $($dirty.Count) file" } else { "bersih" })
$local  = (git rev-parse HEAD).Trim()
$remote = (git rev-parse "origin/$Branch" 2>$null)
if ($remote) { $remote = $remote.Trim() } else { $remote = "0000000-belum-di-push" }
Add-Result "Lokal = GitHub (origin/$Branch)" ($local -eq $remote) ("HEAD {0} / origin {1}" -f $local.Substring(0,7), $remote.Substring(0,7))
Log ("Commit terakhir:`n" + ((git log --oneline -10) -join "`n"))

# ---------------------------------------------------------------- 2. Secret
Section "2. Pemeriksaan secret di repo"
$tracked = git ls-files
$bad = $tracked | Where-Object { $_ -match '(^|/)\.env$|\.pem$|(^|/)certs/|postgres-data|(^|/)reference/' }
Add-Result "Tidak ada .env/.pem/certs/reference ter-commit" (-not $bad) $(if ($bad) { ($bad -join ", ") } else { "aman" })
$history = git log --all --name-only --format="" | Where-Object { $_ -match '(^|/)\.env$|\.pem$|postgres-data' } | Sort-Object -Unique
Add-Result "Riwayat Git bebas .env/.pem" (-not $history) $(if ($history) { ($history -join ", ") } else { "aman" })
$keyLeak = git grep -nIE "DUITKU_API_KEY=[A-Za-z0-9]{8,}|BEGIN (RSA )?PRIVATE KEY" -- . ":(exclude)*.example" 2>$null
Add-Result "Tidak ada API key/private key di kode" (-not $keyLeak) $(if ($keyLeak) { "Temuan: " + ($keyLeak | Select-Object -First 3) -join " | " } else { "aman" })

# ---------------------------------------------------------------- 3. Go modules
Section "3. Go modules: tidy, build, vet, test"
$modules = @("pkg") + (Get-ChildItem services -Directory | Where-Object { Test-Path (Join-Path $_.FullName "go.mod") } | ForEach-Object { "services/$($_.Name)" })
foreach ($m in $modules) {
    if ($goLocal) {
        Push-Location $m
        go mod tidy 2>&1 | Out-Null
        Pop-Location
        git diff --quiet -- "$m/go.mod" "$m/go.sum"
        $tidyOk = ($LASTEXITCODE -eq 0)
        if (-not $tidyOk) { git checkout -- "$m/go.mod" "$m/go.sum" 2>$null }
        Push-Location $m
        $out = $(go build ./... 2>&1; go vet ./... 2>&1; go test -count=1 ./... 2>&1) | Out-String
        $testOk = ($LASTEXITCODE -eq 0) -and ($out -notmatch "FAIL|cannot|undefined|missing go.sum")
        Pop-Location
    } else {
        $cmd = "cd /app/$m && cp go.mod /tmp/m && cp go.sum /tmp/s 2>/dev/null; go mod tidy >/dev/null 2>&1; T=0; cmp -s go.mod /tmp/m && (cmp -s go.sum /tmp/s || [ ! -f /tmp/s ]) || T=1; cp /tmp/m go.mod; [ -f /tmp/s ] && cp /tmp/s go.sum; go build ./... && go vet ./... && go test -count=1 ./... ; R=`$?; echo TIDY=`$T; exit `$R"
        $out = docker run --rm -e GOWORK=off -e GOFLAGS=-buildvcs=false -v "${Root}:/app" -v pmb-gomod:/go/pkg/mod $GoImage sh -c $cmd 2>&1 | Out-String
        $testOk = ($LASTEXITCODE -eq 0)
        $tidyOk = ($out -match "TIDY=0")
    }
    Log "--- $m ---`n$out"
    Add-Result "go mod tidy bersih: $m" $tidyOk $(if ($tidyOk) { "go.mod/go.sum konsisten" } else { "go.mod/go.sum perlu 'go mod tidy'" })
    Add-Result "build/vet/test: $m" $testOk $(if ($testOk) { "lulus" } else { "lihat detail di report" })
}

$ocr = "services/ocr-service"
if (Test-Path $ocr) {
    $pyOut = docker run --rm -v "${Root}/${ocr}:/app" -w /app python:3.12-slim sh -c "pip install -q -r requirements.txt pytest >/dev/null 2>&1 && python -m pytest -q" 2>&1 | Out-String
    $pyOk = ($LASTEXITCODE -eq 0)
    Log "--- ocr-service pytest ---`n$pyOut"
    Add-Result "pytest: ocr-service" $pyOk $(if ($pyOk) { "lulus" } else { "lihat detail di report" })
}

# ---------------------------------------------------------------- 4. Docker dari nol
Section "4. Docker compose dari database kosong"
docker compose -f $Compose down -v 2>&1 | ForEach-Object { Log $_ }
$build = docker compose -f $Compose up -d --build 2>&1 | Out-String
Log $build
Add-Result "docker compose up --build" ($LASTEXITCODE -eq 0) $(if ($LASTEXITCODE -eq 0) { "build & start sukses" } else { "build gagal, lihat report" })

$deadline = (Get-Date).AddMinutes(6)
do {
    Start-Sleep -Seconds 10
    $raw = docker compose -f $Compose ps --format json 2>$null
    $items = @()
    foreach ($line in ($raw -split "`n")) { if ($line.Trim()) { try { $items += ($line | ConvertFrom-Json) } catch {} } }
    $notHealthy = $items | Where-Object { $_.Health -ne "healthy" }
} while ($notHealthy -and (Get-Date) -lt $deadline)
$total = $items.Count
Log ((docker compose -f $Compose ps) -join "`n")
Add-Result "Semua container healthy" ((-not $notHealthy) -and $total -gt 0) $(if ($notHealthy) { "Belum healthy: " + (($notHealthy | ForEach-Object { "$($_.Service)=$($_.State)/$($_.Health)" }) -join ", ") } else { "$total container healthy" })

# ---------------------------------------------------------------- 5. Health via gateway & bypass
Section "5. Health & keamanan dasar"
try { $h = Invoke-WebRequest -UseBasicParsing http://localhost:8000/health/ready -TimeoutSec 15; $hOk = ($h.StatusCode -eq 200) } catch { $hOk = $false }
Add-Result "Gateway /health/ready" $hOk "http://localhost:8000/health/ready"
try { Invoke-WebRequest -UseBasicParsing http://localhost:8000/api/v1/auth/internal/users/x -TimeoutSec 10 | Out-Null; $intBlocked = $false } catch { $intBlocked = $true }
Add-Result "/internal/* diblokir gateway" $intBlocked ""

# ---------------------------------------------------------------- 6. E2E
Section "6. E2E penuh (scripts/e2e.sh)"
if ($gitBashOk -and (-not $notHealthy)) {
    $e2e = & $GitBash scripts/e2e.sh 2>&1 | Out-String
    Log $e2e
    $m = [regex]::Matches($e2e, "PASS:\s*(\d+),\s*FAIL:\s*(\d+)")
    if ($m.Count -gt 0) {
        $pass = [int]$m[$m.Count-1].Groups[1].Value; $fail = [int]$m[$m.Count-1].Groups[2].Value
        Add-Result "E2E" ($fail -eq 0 -and $pass -gt 0) "PASS $pass, FAIL $fail"
        $fails = ($e2e -split "`n") | Where-Object { $_ -match "\[FAIL\]" } | Select-Object -First 15
        if ($fails) { Log ("Baris FAIL:`n" + ($fails -join "`n")) }
    } else {
        Add-Result "E2E" $false "Ringkasan PASS/FAIL tidak ditemukan (script berhenti di tengah?)"
    }
} else {
    Add-Result "E2E" $false "Dilewati: Git Bash tidak ada atau container belum healthy"
}

# ---------------------------------------------------------------- 7. Kelengkapan dokumen
Section "7. Kelengkapan dokumentasi"
$docs = @("docs/contracts","docs/API.md","docs/DECISIONS.md","docs/CHANGE_REQUESTS.md","docs/AUDIT-REPORT.md","docs/FE-BE-GAP.md","docs/PMB.postman_collection.json","docs/ocr-benchmark.md","docs/DUITKU-SANDBOX.md","scripts/duitku-sandbox-check.ps1","docs/RBAC.md")
$docs += @("docs/BUGS.md","docs/DUITKU-PRODUKSI.md","docs/DEPLOY.md","docs/RELEASE-CHECKLIST.md","docs/PANDUAN-ALUR.md","scripts/smoke-camaba.sh","deploy/docker-compose.prod.yml")
if ($Tahap -ne "rilis") { $docs += @("docs/ARCHITECTURE.md","docs/CONSOLIDATION-REPORT.md","docs/SSO.md") }
if ($Tahap -eq "penuh") { $docs += "docs/REFERRAL.md" }
foreach ($d in $docs) { Add-Result "Dokumen $d" (Test-Path $d) "" }
$envText = if (Test-Path ".env.example") { Get-Content ".env.example" -Raw } else { "" }
$envOk = ($envText -match "DUITKU_ENV") -and ($envText -match "APP_ENV")
Add-Result ".env.example memuat DUITKU_ENV & APP_ENV" $envOk ""
$bugText = if (Test-Path "docs/BUGS.md") { Get-Content "docs/BUGS.md" } else { @() }
$openBugs = @($bugText | Where-Object { ($_ -match "\|\s*P[01]\s*\|") -and ($_ -match "\|\s*Open\s*\|") })
Add-Result "BUGS.md tanpa P0/P1 Open" ($openBugs.Count -eq 0) $(if ($openBugs.Count -eq 0) { "aman" } else { "$($openBugs.Count) bug P0/P1 masih Open" })
$svcDirs = Get-ChildItem services -Directory | Where-Object { -not $_.Name.StartsWith("_") } | ForEach-Object { $_.Name } | Sort-Object
if ($Tahap -ne "rilis") {
    $expected = @("gateway","ocr-service","pmb-service")
    $svcOk = (($svcDirs -join ",") -eq ($expected -join ","))
    Add-Result "Jumlah service = 3 (gateway, pmb-service, ocr-service)" $svcOk ("ditemukan: " + ($svcDirs -join ", "))
} else {
    Log ("Service saat ini (" + $svcDirs.Count + "): " + ($svcDirs -join ", "))
}
$svcs = $svcDirs | ForEach-Object { $_ -replace "-service$","" }
foreach ($s in $svcs) {
    $cd = Test-Path "docs/class-diagram-$s.md"; $erd = (Test-Path "docs/erd-$s.md") -or ($s -eq "ocr")
    Add-Result "Class diagram & ERD: $s" ($cd -and $erd) $(if ($cd -and $erd) { "ada" } else { "kurang: " + $(if (-not $cd) { "class-diagram " }) + $(if (-not $erd) { "erd" }) })
}
$crText = if (Test-Path "docs/CHANGE_REQUESTS.md") { Get-Content "docs/CHANGE_REQUESTS.md" -Raw } else { "" }
$required = @("CR-001","CR-002","CR-003","CR-004","CR-005","CR-006","CR-007","CR-008","CR-009","CR-010","CR-013")
$skip = @("CR-011","CR-012","CR-014")
if ($Tahap -eq "konsolidasi") { $required += @("CR-011","CR-012"); $skip = @("CR-014") }
if ($Tahap -eq "penuh")   { $required += @("CR-011","CR-012","CR-014"); $skip = @() }
$found = [regex]::Matches($crText, "CR-\d{3}") | ForEach-Object { $_.Value }
$allCr = ($required + $found) | Sort-Object -Unique | Where-Object { $skip -notcontains $_ }
foreach ($cr in $allCr) {
    $done = $crText -match "$cr[^\n]*(done|selesai)"
    Add-Result "Status $cr = done" $done $(if ($crText -match $cr) { "" } else { "belum dicatat di CHANGE_REQUESTS.md" })
}
$gapText = if (Test-Path "docs/FE-BE-GAP.md") { Get-Content "docs/FE-BE-GAP.md" -Raw } else { "" }
$openRows = ([regex]::Matches($gapText, "\|\s*Open\s*\|")).Count
Add-Result "FE-BE-GAP tanpa baris Open" ($openRows -eq 0) $(if ($openRows -eq 0) { "semua baris selesai/menunggu desain" } else { "$openRows baris masih Open" })
$tags = @("v0.2.0-fase2","v0.5.1-rilis")
if ($Tahap -ne "rilis") { $tags += @("v0.6.0-3-services","v0.7.0-sso-ready") }
if ($Tahap -eq "penuh") { $tags += "v0.8.0-referal" }
git fetch --tags origin *> $null
foreach ($t in $tags) {
    $tagOk = [bool](git tag --list $t)
    Add-Result "Tag $t" $tagOk $(if ($tagOk) { "ada" } else { "belum dibuat (dibuat agent bila DoD terpenuhi)" })
}

# ---------------------------------------------------------------- Ringkasan
$failed = $Results | Where-Object { $_.Status -eq "GAGAL" }
$verdict = if (-not $failed) { "AMAN - SEMUA PEMERIKSAAN LULUS" } else { "TIDAK AMAN - $($failed.Count) PEMERIKSAAN GAGAL" }
Write-Host "`n==================== RINGKASAN ====================" -ForegroundColor Yellow
$Results | Format-Table -AutoSize | Out-String -Width 220 | Write-Host
Write-Host $verdict -ForegroundColor $(if ($failed) { "Red" } else { "Green" })

$header = @(
  "PMB FINAL CHECK REPORT",
  "Tanggal : $(Get-Date -Format 'yyyy-MM-dd HH:mm')",
  "Tahap   : $Tahap",
  "Branch  : $Branch",
  "Commit  : $local",
  "Hasil   : $verdict",
  "",
  ($Results | Format-Table -AutoSize | Out-String -Width 220)
)
($header + "`n==================== DETAIL ====================" + $Log) | Out-File -Encoding utf8 $Report
Write-Host "`nLaporan lengkap: $Report  (kirim file ini ke Claude)" -ForegroundColor Cyan
if ($failed) { exit 1 } else { exit 0 }
