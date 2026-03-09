-- 1. Buat Tabel Users (Nasabah)
CREATE TABLE users (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Buat Tabel Transactions (Riwayat Transfer)
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    sender_id VARCHAR(50) REFERENCES users(id),
    receiver_id VARCHAR(50) REFERENCES users(id),
    amount BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, success, failed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Masukkan Data Dummy (Sesuai dengan ID yang kamu test di cURL tadi)
INSERT INTO users (id, name, balance) VALUES
('123', 'Haris Muhasibi', 10000000),   -- Saldo awal: Rp 10 Juta
('456', 'Fathur Sakha', 5000000),      -- Saldo awal: Rp 5 Juta
('789', 'Dosen Penguji', 1000000);     -- Saldo awal: Rp 1 Juta