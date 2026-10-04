import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up ke 20 VUs
    { duration: '1m', target: 50 },   // Tingkatkan ke 50 VUs
    { duration: '30s', target: 100 },  // Beban puncak 100 VUs
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1500'], // 95% request di bawah 1.5 detik
    http_req_failed: ['rate<0.01'],    // Kegagalan request < 1%
  },
};

const BASE_URL = __ENV.GATEWAY_URL || 'http://localhost:8000';

export default function () {
  // 1. Cek Health Gateway
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'gateway health 200': (r) => r.status === 200,
  });

  // 2. Ambil Periode Pendaftaran Aktif
  const periodsRes = http.get(`${BASE_URL}/api/v1/admission/periods/active`);
  check(periodsRes, {
    'periods active 200': (r) => r.status === 200,
  });

  // 3. Ambil Daftar Jalur Pendaftaran
  const pathsRes = http.get(`${BASE_URL}/api/v1/admission/paths`);
  check(pathsRes, {
    'paths 200': (r) => r.status === 200,
  });

  // 4. Ambil Daftar Program Studi
  const prodisRes = http.get(`${BASE_URL}/api/v1/organization/programs`);
  check(prodisRes, {
    'programs 200': (r) => r.status === 200,
  });

  // 5. Ambil Konfigurasi Pembayaran (Duitku POP)
  const configRes = http.get(`${BASE_URL}/api/v1/payment-config`);
  check(configRes, {
    'payment config 200': (r) => r.status === 200,
  });

  // 6. Ambil Kanal Metode Pembayaran
  const methodsRes = http.get(`${BASE_URL}/api/v1/finance/payment-methods?amount=250000`);
  check(methodsRes, {
    'payment methods 200': (r) => r.status === 200,
  });

  sleep(1);
}
