import http from 'k6/http';
import { check, sleep } from 'k6';

// 1. Konfigurasi "Siksaan" (Virtual Users & Durasi)
export let options = {
    stages: [
        { duration: '10s', target: 50 },  // Pemanasan: Naik pelan-pelan ke 50 user dalam 10 detik
        { duration: '20s', target: 200 }, // PEAK LOAD: Tiba-tiba melonjak ke 200 user yang menembak bersamaan!
        { duration: '10s', target: 0 },   // Pendinginan: Turun perlahan ke 0 user
    ],
};

// 2. Skenario yang dilakukan oleh setiap Virtual User
export default function () {
    const url = 'http://localhost:8081/api/v1/transfer';
    
    // Payload Data Dummy (Haris transfer Rp 10.000 ke Fathur)
    const payload = JSON.stringify({
        sender_id: '123',
        receiver_id: '456',
        amount: 10000,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    // Tembak API-nya!
    let res = http.post(url, payload, params);

    // 3. Validasi (SLA / Service Level Agreement)
    check(res, {
        'Status is 202 (Accepted/Pending)': (r) => r.status === 202,
        'Response time is < 50ms': (r) => r.timings.duration < 50, // API harus merespon di bawah 50 milidetik!
    });

    // Jeda sejenak antar tembakan (simulasi orang ngetik di HP)
    sleep(0.1); 
}