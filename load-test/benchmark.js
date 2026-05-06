import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

// =====================================================
// BENCHMARK TEST — Skenario sama untuk Baseline & Optimized
// Jalankan:
//   k6 run --out json=hasil-baseline.json  load-test/benchmark.js
//   k6 run --out json=hasil-optimized.json load-test/benchmark.js
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
        http_req_failed: ['rate<0.05'],
    },
};

export default function () {
    const BASE_URL = 'http://localhost';
    const randomIP = `192.168.${randomIntBetween(1, 255)}.${randomIntBetween(1, 255)}`;
    const senderNo  = `777${String(randomIntBetween(1, 10000)).padStart(7, '0')}`;
    const receiverNo = `777${String(randomIntBetween(1, 10000)).padStart(7, '0')}`;

    const headers = {
        'X-Simulated-IP': randomIP,
    };

    // 1. CEK SALDO
    let resGet = http.get(`${BASE_URL}/users/${senderNo}/balance`, { headers, tags: { endpoint: 'cek_saldo' } });
    check(resGet, { 'Cek Saldo OK': (r) => r.status === 200 });

    // 2. TRANSFER
    const payload = JSON.stringify({
        sender_account: senderNo,
        receiver_account: receiverNo,
        amount: randomIntBetween(10000, 100000),
        keterangan: 'Benchmark test'
    });

    let resPost = http.post(`${BASE_URL}/api/v1/transfer`, payload, {
        headers: { ...headers, 'Content-Type': 'application/json' },
        tags: { endpoint: 'transfer' }
    });
    check(resPost, { 'Transfer OK': (r) => r.status === 200 || r.status === 202 });

    sleep(0.1);
}

// Export hasil ke file text untuk mudah dibaca
export function handleSummary(data) {
    // Ambil metrik penting
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
