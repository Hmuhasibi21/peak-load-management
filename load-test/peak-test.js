import http from 'k6/http';
import { check } from 'k6';

export const options = {
    scenarios: {
        satu_juta_transaksi: {
            executor: 'constant-arrival-rate',
            rate: 278,          // 278 RPS (Target 1 Juta / Jam)
            timeUnit: '1s',
            duration: '1m',     // Coba tes selama 1 menit dulu
            preAllocatedVUs: 50,
            maxVUs: 500,
        },
    },
    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
    },
};

// Fungsi pembuat IP Acak (Misal: 192.168.1.45)
function getRandomIP() {
    return `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`;
}

export default function () {
    const randomUserID = Math.floor(Math.random() * 1000);
    const urlTransfer = 'http://localhost:8080/api/v1/transfer';
    const urlBalance = `http://localhost:8080/users/user_${randomUserID}/balance`;

    const payload = JSON.stringify({
        sender_id: `user_${randomUserID}`,
        receiver_id: `user_${Math.floor(Math.random() * 1000)}`,
        amount: 50000.0,
    });

    // MASUKKAN IP ACAK KE DALAM HEADER (KTP PALSU)
    const params = {
        headers: { 
            'Content-Type': 'application/json',
            'X-Simulated-IP': getRandomIP() 
        },
    };

    // Eksekusi API
    let resBalance = http.get(urlBalance, params);
    check(resBalance, { 'Cek Saldo 200': (r) => r.status === 200 });

    let resTransfer = http.post(urlTransfer, payload, params);
    check(resTransfer, { 'Transfer 202 (Masuk Antrean)': (r) => r.status === 202 });
}