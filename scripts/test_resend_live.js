const net = require('net');

function delKey(key) {
  return new Promise((resolve) => {
    const s = net.connect(6379, '127.0.0.1', () => {
      s.write('*2\r\n$3\r\nDEL\r\n$' + key.length + '\r\n' + key + '\r\n');
    });
    s.on('data', () => { s.end(); resolve(); });
    s.on('error', () => { resolve(); });
  });
}

async function main() {
  const timestamp = Date.now();
  const email = `calon_mhs_${timestamp}@pmb.bth.ac.id`;
  console.log(`\n=== 1. REGISTRASI CALON MAHASISWA BARU (${email}) ===`);

  const regRes = await fetch('http://127.0.0.1:8000/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      full_name: 'Pendaftar Uji Mailpit Live',
      email: email,
      phone: '081399887766',
      password: 'PasswordBTH2026!',
      password_confirmation: 'PasswordBTH2026!'
    })
  });
  const regJson = await regRes.json();
  console.log('Status Register:', regRes.status, regJson.message);

  // Tunggu email masuk
  await new Promise(r => setTimeout(r, 1500));
  let mRes = await fetch('http://127.0.0.1:8025/api/v1/messages');
  let mData = await mRes.json();
  let msgs = mData.messages.filter(m => m.To[0].Address === email);
  console.log('Jumlah email di Mailpit setelah Register:', msgs.length);
  if (msgs.length > 0) {
    console.log('ID Email #1:', msgs[0].ID, '| Subject:', msgs[0].Subject);
  }

  console.log(`\n=== 2. MENGUJI COOLDOWN RESEND OTP ===`);
  const cdRes = await fetch('http://127.0.0.1:8000/api/v1/auth/resend-otp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email })
  });
  const cdJson = await cdRes.json();
  console.log('Status Resend saat Cooldown:', cdRes.status, cdJson.error_code, '-', cdJson.message);

  console.log(`\n=== 3. RESET COOLDOWN & KIRIM ULANG KODE OTP ===`);
  await delKey('auth:cooldown:resend:' + email);
  console.log('Cooldown dihapus dari Redis...');

  const resendRes = await fetch('http://127.0.0.1:8000/api/v1/auth/resend-otp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email })
  });
  const resendJson = await resendRes.json();
  console.log('Status Resend Baru:', resendRes.status, resendJson.message);

  await new Promise(r => setTimeout(r, 1500));
  mRes = await fetch('http://127.0.0.1:8025/api/v1/messages');
  mData = await mRes.json();
  msgs = mData.messages.filter(m => m.To[0].Address === email);
  console.log('Jumlah email di Mailpit setelah Resend OTP:', msgs.length);
  msgs.forEach((m, i) => {
    console.log(`- Email #${i+1}: ID=${m.ID} | Waktu=${m.Created}`);
  });

  if (msgs.length >= 2) {
    console.log('\n🎉 KESIMPULAN: KIRIM ULANG KODE OTP BERHASIL 100% MASUK KE MAILPIT!');
  } else {
    console.log('\n❌ KESIMPULAN: Email kedua belum ditemukan di Mailpit.');
  }

  console.log(`\n=== 4. MENGUJI RESEND OTP PADA AKUN SUDAH AKTIF ===`);
  // Verifikasi akun tersebut dengan OTP terbaru
  const latestMsgId = msgs[0].ID;
  const msgDetailRes = await fetch(`http://127.0.0.1:8025/api/v1/message/${latestMsgId}`);
  const msgDetail = await msgDetailRes.json();
  const otpMatch = msgDetail.HTML.match(/\b\d{6}\b/);
  const otpCode = otpMatch ? otpMatch[0] : '';
  console.log('Kode OTP yang diambil dari email Mailpit:', otpCode);

  const verifyRes = await fetch('http://127.0.0.1:8000/api/v1/auth/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, otp: otpCode })
  });
  console.log('Status Verifikasi Akun:', verifyRes.status, (await verifyRes.json()).message);

  // Coba resend lagi setelah akun aktif
  await delKey('auth:cooldown:resend:' + email);
  const resendActiveRes = await fetch('http://127.0.0.1:8000/api/v1/auth/resend-otp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email })
  });
  const resendActiveJson = await resendActiveRes.json();
  console.log('Resend OTP pada akun aktif:', resendActiveRes.status, resendActiveJson.error_code, '-', resendActiveJson.message);
}

main().catch(console.error);
