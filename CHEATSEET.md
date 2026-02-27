# 🚀 CHEATSHEET & SETUP COMMANDS - ELASTICSIX

Dokumen ini berisi kumpulan perintah penting untuk menjalankan, memodifikasi, dan menguji project Peak Load Management System (Capstone Bank A).

---

## 👥 1. Perintah Dasar Git (Untuk Semua Anggota Tim)

Gunakan perintah ini untuk sinkronisasi kode dengan GitHub.

- **Clone Repository (Pertama kali):**
  `git clone <link-repo-github-elasticsix>`
- **Menyimpan perubahan dan Push ke GitHub:**
  `git add .`
  `git commit -m "pesan perubahan kamu, contoh: feat: add redis cache"`
  `git push origin main`
- **Mengambil update terbaru dari GitHub (Wajib sebelum ngoding!):**
  `git pull origin main`

---

## 🐳 2. Infrastruktur & Docker (Tim Tekkom: Haris, Gilang, Oxzy)

Pastikan Docker Desktop sudah menyala sebelum menjalankan perintah ini.

- **Menyalakan Semua Server (DB, Redis, MQ, API, Grafana):**
  `docker compose up -d --build`
- **Mematikan Semua Server:**
  `docker compose down`
- **Melihat Status Server (Apakah ada yang mati/crash?):**
  `docker compose ps`
- **Melihat Log Aplikasi (Untuk cek error dari API Golang):**
  `docker logs elasticsix_api -f`
- **Melihat Log Database PostgreSQL:**
  `docker logs elasticsix_postgres -f`

---

## 🐹 3. Golang & Setup Library (Tim TIF: Fathur)

Perintah ini dijalankan jika ingin menambah fitur baru di backend. Pastikan sudah menginstal Golang di laptop (versi 1.21+).

- **Inisialisasi Project (Sudah dilakukan Haris):**
  `go mod init peak-load-management`
- **Install Framework Go Fiber (Web Framework Utama):**
  `go get -u github.com/gofiber/fiber/v2`
- **Install Driver PostgreSQL (GORM / PGX):**
  `go get -u gorm.io/gorm`
  `go get -u gorm.io/driver/postgres`
- **Install Driver Redis:**
  `go get -u github.com/redis/go-redis/v9`
- **Install Driver RabbitMQ:**
  `go get -u github.com/rabbitmq/amqp091-go`
- **Merapikan & Download Library yang kurang:**
  `go mod tidy`

---

## 🔫 4. Load Testing dengan k6 (Tim Tekkom / SRE)

Perintah untuk menyiksa server dan melihat apakah SLA (Latency < 200ms) tercapai.

- **Install k6 (Mac Users):**
  `brew install k6`
- **Install k6 (Windows Users):**
  `winget install k6`
- **Jalankan Skenario Baseline (Tes Pemanasan):**
  `k6 run load-test/baseline.js`
- **Jalankan Skenario Peak Load (Nanti dibuat saat API selesai):**
  `k6 run load-test/peak-test.js`

_(SRE Hack: Jika k6 belum terinstall di laptop, bisa jalankan k6 via Docker)_:
`docker run --rm -i -v "$(pwd):/app" -w /app grafana/k6 run load-test/baseline.js`

---

## 🌐 5. Daftar Tautan Penting (Akses via Browser)

Saat `docker compose up -d` sudah hijau semua, akses link ini:

1. **API Golang (Pintu Masuk):** `http://localhost:8080/ping`
2. **RabbitMQ Dashboard (Antrean Transaksi):** `http://localhost:15672` (User: `admin`, Pass: `admin123`)
3. **Prometheus (Data Scraper):** `http://localhost:9090`
4. **Grafana (Monitoring Utama untuk Demo):** `http://localhost:3000` (User: `admin`, Pass: `admin123`)
5. **Database PostgreSQL:** Buka DBeaver / PgAdmin -> host: `localhost`, port: `5432`, user: `root`, pass: `password`, db: `bank_a_db`.
