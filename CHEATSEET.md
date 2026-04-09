# 🚀 ELASTICSIX SRE CHEATSHEET & PLAYBOOK
*Dokumen panduan operasional harian untuk sistem Microservices Bank A.*

---

## 🏗️ 1. RINGKASAN ARSITEKTUR SISTEM
Sistem ini dirancang kebal terhadap lonjakan *traffic* (Peak Load) dengan komponen berikut:
* **Load Balancer & Gateway:** NGINX (Meneruskan *traffic* ke API).
* **API Service:** Golang (Fiber) dengan *Rate Limiter* & *Prometheus Middleware*.
* **Database (Read/Write Split):** PostgreSQL Master (Write) & PostgreSQL Replica (Read).
* **Caching:** Redis (Mempercepat pengecekan saldo).
* **Message Queue / Asynchronous:** RabbitMQ (Mengantrekan transaksi transfer agar DB tidak *crash*).
* **Keamanan Ekstra:** Circuit Breaker (Memutus koneksi DB jika terjadi *timeout*).
* **Monitoring & Observability:** Prometheus & Grafana.

---

## 💻 2. PERINTAH SAKTI DOCKER (JALUR SRE)
Sebagai SRE, kita **TIDAK LAGI** menggunakan `go run main.go`. Biarkan Docker yang mengurus semuanya di latar belakang.

* **Menyalakan Seluruh Sistem (Auto-Build & Background):**
    ```bash
    docker compose up --build -d
    ```
* **Mematikan Seluruh Sistem:**
    ```bash
    docker compose down
    ```
* **Melihat Status Services (Apakah sudah "Up"?):**
    ```bash
    docker ps
    ```
* **Melihat Log Error dari API Golang:**
    ```bash
    docker logs elasticsix_api -f
    ```

---

## 🔗 3. URL & CREDENTIALS PENTING

| Layanan | URL Lokal (Browser) | Username | Password |
| :--- | :--- | :--- | :--- |
| **API / NGINX** | `http://localhost` | - | - |
| **Grafana** | `http://localhost:3000` | `admin` | `admin123` |
| **RabbitMQ UI** | `http://localhost:15672` | `admin` | `admin123` |
| **Prometheus** | `http://localhost:9090` | - | - |

> **🔥 CATATAN PENTING GRAFANA:** > Saat setting *Data Source* Prometheus di dalam Grafana, gunakan URL **`http://prometheus:9090`** (bukan localhost), karena Grafana harus memanggil *container* Prometheus lewat jalur internal Docker.

---

## 🔫 4. LOAD TESTING (K6)
Gunakan perintah ini untuk mensimulasikan gempuran puluhan ribu pengguna secara bersamaan.

* **Perintah Eksekusi:**
    ```bash
    k6 run peak-test.js
    ```
* **Syarat k6 Berhasil (Anti-Block):**
    * Wajib menyisipkan header `Content-Type: application/json` pada request POST.
    * Wajib menyisipkan header `X-Simulated-IP: 192.168.x.x` (IP Acak) untuk menghindari blokir *Rate Limiter* NGINX/Golang.

---

## 🌐 5. DAFTAR ENDPOINT API & CURL TEST

**1. Cek Kesehatan Server (Health Check)**
```bash
curl -I http://localhost/ping