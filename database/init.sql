-- =====================================================
-- SCHEMA DATABASE BANK A - ElasticSix
-- Data nasabah mirip bank asli Indonesia
-- =====================================================

-- 1. Tabel Nasabah (Data Lengkap Nasabah Bank)
CREATE TABLE nasabah (
    id SERIAL PRIMARY KEY,
    nomor_rekening VARCHAR(20) UNIQUE NOT NULL,
    nama_lengkap VARCHAR(100) NOT NULL,
    nik VARCHAR(16) UNIQUE NOT NULL,
    tempat_lahir VARCHAR(50),
    tanggal_lahir DATE,
    jenis_kelamin VARCHAR(10) DEFAULT 'L',           -- L / P
    nomor_telepon VARCHAR(15),
    email VARCHAR(100),
    alamat TEXT,
    kota VARCHAR(50),
    kode_pos VARCHAR(5),
    saldo BIGINT NOT NULL DEFAULT 0,
    jenis_rekening VARCHAR(20) DEFAULT 'tabungan',    -- tabungan / giro
    status_rekening VARCHAR(20) DEFAULT 'aktif',      -- aktif / blokir / dormant / tutup
    cabang VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabel Transaksi (Riwayat Transfer Antar Rekening)
CREATE TABLE transaksi (
    id SERIAL PRIMARY KEY,
    nomor_referensi VARCHAR(30) UNIQUE NOT NULL,
    rekening_pengirim VARCHAR(20) REFERENCES nasabah(nomor_rekening),
    rekening_penerima VARCHAR(20) REFERENCES nasabah(nomor_rekening),
    jumlah BIGINT NOT NULL,
    jenis_transaksi VARCHAR(20) NOT NULL DEFAULT 'transfer',  -- transfer / setor / tarik
    status VARCHAR(20) DEFAULT 'pending',                      -- pending / berhasil / gagal
    keterangan TEXT,
    biaya_admin BIGINT DEFAULT 0,
    saldo_pengirim_sebelum BIGINT,
    saldo_pengirim_sesudah BIGINT,
    channel VARCHAR(30) DEFAULT 'mobile_banking',              -- mobile_banking / internet_banking / atm / teller
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Index untuk performa query saat Peak Load
CREATE INDEX idx_nasabah_rekening ON nasabah(nomor_rekening);
CREATE INDEX idx_nasabah_status ON nasabah(status_rekening);
CREATE INDEX idx_transaksi_pengirim ON transaksi(rekening_pengirim);
CREATE INDEX idx_transaksi_penerima ON transaksi(rekening_penerima);
CREATE INDEX idx_transaksi_status ON transaksi(status);

-- =====================================================
-- 4. SEED DATA: 10.000 Nasabah dengan data realistis
-- =====================================================
INSERT INTO nasabah (
    nomor_rekening, nama_lengkap, nik, tempat_lahir, tanggal_lahir,
    jenis_kelamin, nomor_telepon, email, alamat, kota, kode_pos,
    saldo, jenis_rekening, status_rekening, cabang
)
SELECT
    -- Nomor Rekening 10 digit: 7770000001 - 7770002000
    '777' || LPAD(n::text, 7, '0'),

    -- Nama Lengkap (kombinasi nama depan + belakang Indonesia)
    (ARRAY[
        'Ahmad','Budi','Citra','Dewi','Eko','Fajar','Gilang','Hani','Irfan','Joko',
        'Kartika','Lukman','Maya','Nia','Omar','Putri','Rizky','Rina','Sari','Taufik',
        'Umi','Vina','Wahyu','Wulan','Yusuf','Zahra','Andi','Bayu','Dian','Endah',
        'Faisal','Galuh','Hendra','Indra','Jasmine','Kiki','Laras','Nanda','Okta','Surya'
    ])[1 + (n-1) % 40]
    || ' ' ||
    (ARRAY[
        'Pratama','Wijaya','Kusuma','Sari','Hidayat','Nugraha','Permana','Lestari',
        'Saputra','Wibowo','Ramadhani','Putri','Firmansyah','Utami','Setiawan',
        'Maharani','Kurniawan','Anggraini','Prasetyo','Handayani','Santoso','Purnama',
        'Suryadi','Fitriani','Maulana','Dewanti','Hakim','Susanti','Ramadhan','Kusnadi'
    ])[1 + ((n-1) / 40) % 30],

    -- NIK 16 digit (format: provinsi + kota + kecamatan + tgl lahir + urut)
    (ARRAY['3573','3578','3171','3273','3471'])[1 + (n-1) % 5]
    || (ARRAY['01','02','03','04','05','06','07','08','09','10'])[1 + (n-1) % 10]
    || LPAD(((n % 28) + 1)::text, 2, '0')
    || LPAD(((n % 12) + 1)::text, 2, '0')
    || LPAD((90 + (n % 10))::text, 2, '0')
    || LPAD(n::text, 4, '0'),

    -- Tempat Lahir
    (ARRAY['Malang','Surabaya','Jakarta','Bandung','Yogyakarta',
           'Semarang','Denpasar','Medan','Makassar','Palembang'])[1 + (n-1) % 10],

    -- Tanggal Lahir (1990-01-01 sampai 2000-12-31)
    '1990-01-01'::date + ((n * 2) % 3650),

    -- Jenis Kelamin
    CASE WHEN n % 2 = 0 THEN 'P' ELSE 'L' END,

    -- Nomor Telepon (08xx-xxxx-xxxx)
    '08' || (ARRAY['13','21','55','77','58','12','31','78','38','19'])[1 + (n-1) % 10]
    || LPAD((1000000 + n)::text, 7, '0'),

    -- Email
    LOWER(
        (ARRAY[
            'ahmad','budi','citra','dewi','eko','fajar','gilang','hani','irfan','joko',
            'kartika','lukman','maya','nia','omar','putri','rizky','rina','sari','taufik',
            'umi','vina','wahyu','wulan','yusuf','zahra','andi','bayu','dian','endah',
            'faisal','galuh','hendra','indra','jasmine','kiki','laras','nanda','okta','surya'
        ])[1 + (n-1) % 40]
        || '.' ||
        (ARRAY[
            'pratama','wijaya','kusuma','sari','hidayat','nugraha','permana','lestari',
            'saputra','wibowo','ramadhani','putri','firmansyah','utami','setiawan',
            'maharani','kurniawan','anggraini','prasetyo','handayani','santoso','purnama',
            'suryadi','fitriani','maulana','dewanti','hakim','susanti','ramadhan','kusnadi'
        ])[1 + ((n-1) / 40) % 30]
    ) || n || '@gmail.com',

    -- Alamat
    'Jl. ' || (ARRAY[
        'Merdeka','Sudirman','Thamrin','Gatot Subroto','Diponegoro',
        'Ahmad Yani','Pahlawan','Kartini','Imam Bonjol','Hayam Wuruk'
    ])[1 + (n-1) % 10]
    || ' No. ' || ((n % 200) + 1)
    || ', RT ' || LPAD(((n % 10) + 1)::text, 2, '0')
    || '/RW ' || LPAD(((n % 5) + 1)::text, 2, '0'),

    -- Kota
    (ARRAY['Malang','Surabaya','Jakarta Selatan','Bandung','Yogyakarta',
           'Semarang','Denpasar','Medan','Makassar','Palembang'])[1 + (n-1) % 10],

    -- Kode Pos
    (ARRAY['65141','60231','12190','40132','55281',
           '50134','80232','20112','90222','30137'])[1 + (n-1) % 10],

    -- Saldo (Rp 1 juta - Rp 100 juta, random)
    1000000 + (random() * 99000000)::BIGINT,

    -- Jenis Rekening (90% tabungan, 10% giro)
    CASE WHEN n % 10 = 0 THEN 'giro' ELSE 'tabungan' END,

    -- Status Rekening (semua aktif untuk load test)
    'aktif',

    -- Cabang
    (ARRAY[
        'KCP Malang Kota','KCP Malang Lowokwaru','KCP Surabaya Pusat',
        'KCP Jakarta Sudirman','KCP Bandung Dago','KCP Yogyakarta Malioboro',
        'KCP Semarang Simpang Lima','KCP Denpasar Sanur','KCP Medan Merdeka','KCP Makassar Losari'
    ])[1 + (n-1) % 10]

FROM generate_series(1, 10000) AS n;

-- Tampilkan jumlah data
DO $$ BEGIN RAISE NOTICE 'Berhasil membuat % nasabah', (SELECT count(*) FROM nasabah); END $$;