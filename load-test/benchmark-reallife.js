import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

// =====================================================
// BENCHMARK REAL-LIFE — Simulasi User Asli
//
// Setiap IP 100% UNIK (tidak pernah tabrakan)
// Menggunakan kombinasi VU ID + Iteration Counter
// Sehingga Rate Limiter TIDAK akan memblokir siapapun
// =====================================================

export const options = {
    scenarios: {
        benchmark: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '10s', target: 50 },   // Ramp-up
                { duration: '40s', target: 50 },   // Sustained peak
                { duration: '10s', target: 0 },    // Ramp-down
            ],
            gracefulRampDown: '30s',
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<500'],
        http_req_failed: ['rate<0.01'],
    },
};

export default function () {
    const BASE_URL = 'http://localhost';

    // =====================================================
    // IP UNIK per iterasi — TIDAK PERNAH TABRAKAN
    // Format: 10.{VU_ID}.{iter_high}.{iter_low}
    // VU 1 iter 0   → 10.1.0.0
    // VU 1 iter 1   → 10.1.0.1
    // VU 1 iter 256  → 10.1.1.0
    // VU 2 iter 0   → 10.2.0.0
    // Setiap IP hanya dipakai 1x (2 request: GET + POST)
    // = Maksimal 2 req/IP, jauh di bawah limit 30/menit
    // =====================================================
    const vu = __VU;
    const iter = __ITER;
    const octet3 = Math.floor(iter / 256) % 256;
    const octet4 = iter % 256;
    const uniqueIP = `10.${vu}.${octet3}.${octet4}`;

    // Nomor rekening random dari database (7770000001 - 7770002000)
    const senderNo  = `777${String(randomIntBetween(1, 10000)).padStart(7, '0')}`;
    const receiverNo = `777${String(randomIntBetween(1, 10000)).padStart(7, '0')}`;

    const headers = { 'X-Simulated-IP': uniqueIP };

    // 1. CEK SALDO
    let resGet = http.get(`${BASE_URL}/users/${senderNo}/balance`, {
        headers,
        tags: { endpoint: 'cek_saldo' }
    });
    check(resGet, { 'Cek Saldo OK': (r) => r.status === 200 });

    // 2. TRANSFER
    const payload = JSON.stringify({
        sender_account: senderNo,
        receiver_account: receiverNo,
        amount: randomIntBetween(10000, 100000),
        keterangan: 'Transfer real-life benchmark'
    });

    let resPost = http.post(`${BASE_URL}/api/v1/transfer`, payload, {
        headers: { ...headers, 'Content-Type': 'application/json' },
        tags: { endpoint: 'transfer' }
    });
    check(resPost, { 'Transfer OK': (r) => r.status === 200 || r.status === 202 });

    sleep(0.1);
}

export function handleSummary(data) {
    const duration = data.metrics.http_req_duration;
    const reqs = data.metrics.http_reqs;
    const failed = data.metrics.http_req_failed;

    const summary = `
============================================
  HASIL BENCHMARK - ${new Date().toISOString()}
============================================
Total Requests     : ${reqs.values.count}
Throughput (RPS)   : ${reqs.values.rate.toFixed(2)} req/s
Avg Response Time  : ${duration.values.avg.toFixed(2)} ms
Med Response Time  : ${duration.values.med.toFixed(2)} ms
P90 Response Time  : ${duration.values['p(90)'].toFixed(2)} ms
P95 Response Time  : ${duration.values['p(95)'].toFixed(2)} ms
Max Response Time  : ${duration.values.max.toFixed(2)} ms
Error Rate         : ${(failed.values.rate * 100).toFixed(2)}%
============================================
`;

    return {
        stdout: summary + '\n' + textSummary(data, { indent: '  ', enableColors: true }),
    };
}
