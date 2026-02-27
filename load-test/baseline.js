import http from 'k6/http';
import { check, sleep } from 'k6';

// 1. Konfigurasi Skenario Uji Beban (Load Test Options)
export const options = {
  stages: [
    { duration: '10s', target: 50 },  // Fase 1: Naik perlahan dari 0 ke 50 Virtual Users (VU) selama 10 detik
    { duration: '20s', target: 50 },  // Fase 2: Tahan di 50 VU selama 20 detik (Peak Load Simulasi)
    { duration: '10s', target: 0 },   // Fase 3: Turun perlahan kembali ke 0 VU selama 10 detik
  ],
  thresholds: {
    // Kita menetapkan SLA (Service Level Agreement):
    // 95% dari request HARUS selesai di bawah 200ms
    http_req_duration: ['p(95)<200'], 
    // Error rate tidak boleh lebih dari 1%
    http_req_failed: ['rate<0.01'],   
  },
};

// 2. Fungsi Utama yang akan dijalankan oleh setiap Virtual User (Nasabah)
export default function () {
  // Karena API Fathur belum jadi, kita tembak endpoint /ping dulu
    const url = 'http://localhost:8080/ping';
  // const url = 'http://host.docker.internal:8080/ping';
  // Nanti kalau API Fathur sudah jadi, tinggal di-uncomment yang ini:
  // const url = 'http://localhost:8080/api/v1/transaction';
  // const payload = JSON.stringify({ user_id: 123, amount: 50000 });
  // const params = { headers: { 'Content-Type': 'application/json' } };
  // const res = http.post(url, payload, params);

  const res = http.get(url);

  // 3. Validasi (Check)
  // Memastikan server membalas dengan status 200 OK
  check(res, {
    'status adalah 200': (r) => r.status === 200,
  });

  // Jeda 1 detik antar request agar laptopmu tidak langsung hang
  sleep(1);
}