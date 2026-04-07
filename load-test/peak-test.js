import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
    scenarios: {
        satu_juta_transaksi: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '10s', target: 50 }, // Pemanasan naik ke 50 VUs (Virtual Users)
                { duration: '40s', target: 50 }, // Gempur stabil di 50 VUs
                { duration: '10s', target: 0 },  // Pendinginan turun ke 0
            ],
            gracefulRampDown: '30s',
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% request harus di bawah 500ms
        http_req_failed: ['rate<0.01'],   // Error rate tidak boleh lebih dari 1%
    },
};

export default function () {
    // Alamat utama lewat NGINX (Port 80)
    const BASE_URL = 'http://localhost'; 
    
    // [TRICK SRE] Bikin IP acak agar tidak diblokir Rate Limiter Golang (Max 30 req/min/IP)
    const randomIP = `192.168.${randomIntBetween(1, 255)}.${randomIntBetween(1, 255)}`;
    const randomUserID = randomIntBetween(1, 1000);

    // ==============================================
    // 1. SKENARIO GET: CEK SALDO
    // ==============================================
    const getParams = {
        headers: {
            'X-Simulated-IP': randomIP, // Suntikkan IP palsu
        },
    };
    
    let resGet = http.get(`${BASE_URL}/users/${randomUserID}/balance`, getParams);
    check(resGet, {
        '✓ Cek Saldo 200': (r) => r.status === 200,
    });


    // ==============================================
    // 2. SKENARIO POST: TRANSFER UANG
    // ==============================================
    // Format JSON yang wajib sesuai dengan struct TransferRequest di main.go
    const payload = JSON.stringify({
        sender_id: `user_${randomUserID}`,
        receiver_id: `user_${randomIntBetween(1001, 2000)}`,
        amount: randomIntBetween(10000, 500000) // Transfer nominal acak
    });

    const postParams = {
        headers: {
            'Content-Type': 'application/json', // INI WAJIB ADA AGAR GOLANG TIDAK MENOLAK (HTTP 400)
            'X-Simulated-IP': randomIP,         // Suntikkan IP palsu yang sama
        },
    };

    let resPost = http.post(`${BASE_URL}/api/v1/transfer`, payload, postParams);
    check(resPost, {
        '✓ Transfer 202 (Masuk Antrean)': (r) => r.status === 202,
    });

    // Beri jeda sepersekian detik agar sistem mencerna seperti manusia asli
    sleep(0.1); 
}