package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
)

// Global Variables
var dbMaster *sql.DB
var dbReplica *sql.DB
var rdb *redis.Client
var mqConn *amqp.Connection
var mqChannel *amqp.Channel
var ctx = context.Background()
var cb *gobreaker.CircuitBreaker
var jumlahKapasitasServer = 1

// Struct JSON Transfer (Format mirip request transfer bank asli)
type TransferRequest struct {
	SenderAccount   string `json:"sender_account"`
	ReceiverAccount string `json:"receiver_account"`
	Amount          int64  `json:"amount"`
	Keterangan      string `json:"keterangan"`
}

// 1. Init Database (Read/Write Splitting)
func initDB() {
	var err error
	masterURL := os.Getenv("DB_MASTER_URL")
	replicaURL := os.Getenv("DB_REPLICA_URL")

	// Menggunakan localhost agar bisa di-run lokal (go run main.go) di luar Docker
	if masterURL == "" {
		masterURL = "postgres://root:password@localhost:5432/bank_a_db?sslmode=disable"
	}
	if replicaURL == "" {
		replicaURL = "postgres://root:password@localhost:5433/bank_a_db?sslmode=disable" // PORT 5433 (Replica)
	}

	dbMaster, err = sql.Open("postgres", masterURL)
	if err != nil {
		log.Fatal("Gagal konek ke Master DB:", err)
	}
	// Konfigurasi Connection Pool untuk Peak Load
	dbMaster.SetMaxOpenConns(50)
	dbMaster.SetMaxIdleConns(25)
	dbMaster.SetConnMaxLifetime(5 * time.Minute)

	dbReplica, err = sql.Open("postgres", replicaURL)
	if err != nil {
		log.Fatal("Gagal konek ke Replica DB:", err)
	}
	dbReplica.SetMaxOpenConns(50)
	dbReplica.SetMaxIdleConns(25)
	dbReplica.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("🗄️ Berhasil terhubung ke Master dan Replica Database!")
}

// 2. Init Redis Cache
func initRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	fmt.Println("⚡ Redis Client disiapkan!")
}

// 3. Init RabbitMQ (DENGAN RETRY MECHANISM SRE)
func initRabbitMQ() {
	var err error
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://admin:admin123@localhost:5672/"
	}

	// [TRICK SRE] Coba konek maksimal 15 kali, jangan langsung mati kalau gagal
	maxRetries := 15
	for i := 1; i <= maxRetries; i++ {
		mqConn, err = amqp.Dial(rabbitURL)
		if err == nil {
			break // Sukses? Langsung keluar dari looping!
		}
		fmt.Printf("⏳ Menunggu RabbitMQ siap... (Percobaan %d/%d)\n", i, maxRetries)
		time.Sleep(5 * time.Second) // Tunggu 5 detik sebelum coba lagi
	}

	if err != nil {
		log.Fatalf("❌ Gagal konek ke RabbitMQ setelah %d percobaan: %v", maxRetries, err)
	}

	mqChannel, err = mqConn.Channel()
	if err != nil {
		log.Fatal("Gagal buka channel RabbitMQ:", err)
	}

	_, err = mqChannel.QueueDeclare(
		"transfer_queue", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatal("Gagal deklarasi queue RabbitMQ:", err)
	}

	fmt.Println("🐇 RabbitMQ siap menerima antrean transfer!")
}

// 4. Init Circuit Breaker
func initCircuitBreaker() {
	cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "Database_Replica_CB",
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("\n🚨 [CIRCUIT BREAKER] Status berubah dari %s menjadi %s!\n\n", from, to)
		},
	})
	fmt.Println("🛡️ Circuit Breaker (Sekering Anti-Crash) disiapkan!")
}

// =====================================================================
// BACKGROUND WORKER (RABBITMQ CONSUMER) — REAL DB OPERATIONS
// =====================================================================
func startRabbitMQWorkers(numWorkers int) {
	msgs, err := mqChannel.Consume("transfer_queue", "", false, false, false, false, nil)
	if err != nil {
		log.Println("Gagal consume RabbitMQ:", err)
		return
	}

	fmt.Printf("👷 Menyiapkan %d Pekerja Transaksi di latar belakang...\n", numWorkers)
	for i := 1; i <= numWorkers; i++ {
		go func(workerID int) {
			for d := range msgs {
				var req TransferRequest
				if err := json.Unmarshal(d.Body, &req); err != nil {
					log.Printf("[Worker %d] ❌ Gagal parse JSON: %v", workerID, err)
					d.Nack(false, false)
					continue
				}

				// Mulai Database Transaction (ACID)
				tx, err := dbMaster.Begin()
				if err != nil {
					log.Printf("[Worker %d] ❌ Gagal mulai transaksi DB: %v", workerID, err)
					d.Nack(false, true) // Requeue agar dicoba ulang
					continue
				}

				// 1. Kunci baris pengirim (Pessimistic Locking) & cek saldo
				var saldoPengirim int64
				err = tx.QueryRow(
					"SELECT saldo FROM nasabah WHERE nomor_rekening = $1 AND status_rekening = 'aktif' FOR UPDATE",
					req.SenderAccount,
				).Scan(&saldoPengirim)

				if err != nil {
					tx.Rollback()
					d.Ack(false) // Rekening tidak ditemukan, jangan requeue
					continue
				}

				// 2. Validasi saldo mencukupi
				if saldoPengirim < req.Amount {
					tx.Rollback()
					d.Ack(false) // Saldo tidak cukup, jangan requeue
					continue
				}

				// 3. Kurangi saldo pengirim
				_, err = tx.Exec(
					"UPDATE nasabah SET saldo = saldo - $1, updated_at = NOW() WHERE nomor_rekening = $2",
					req.Amount, req.SenderAccount,
				)
				if err != nil {
					tx.Rollback()
					d.Nack(false, true)
					continue
				}

				// 4. Tambah saldo penerima
				_, err = tx.Exec(
					"UPDATE nasabah SET saldo = saldo + $1, updated_at = NOW() WHERE nomor_rekening = $2",
					req.Amount, req.ReceiverAccount,
				)
				if err != nil {
					tx.Rollback()
					d.Nack(false, true)
					continue
				}

				// 5. Catat riwayat transaksi
				noRef := fmt.Sprintf("TRX%d%04d", time.Now().UnixNano(), workerID)
				keterangan := req.Keterangan
				if keterangan == "" {
					keterangan = "Transfer antar rekening"
				}
				_, err = tx.Exec(
					`INSERT INTO transaksi 
					(nomor_referensi, rekening_pengirim, rekening_penerima, jumlah, 
					 jenis_transaksi, status, keterangan, saldo_pengirim_sebelum, saldo_pengirim_sesudah)
					VALUES ($1, $2, $3, $4, 'transfer', 'berhasil', $5, $6, $7)`,
					noRef, req.SenderAccount, req.ReceiverAccount, req.Amount,
					keterangan, saldoPengirim, saldoPengirim-req.Amount,
				)
				if err != nil {
					tx.Rollback()
					d.Nack(false, true)
					continue
				}

				// 6. COMMIT — semua operasi berhasil!
				if err = tx.Commit(); err != nil {
					d.Nack(false, true)
					continue
				}

				// 7. Hapus cache Redis agar saldo terbaru terbaca
				rdb.Del(ctx, "saldo:"+req.SenderAccount)
				rdb.Del(ctx, "saldo:"+req.ReceiverAccount)

				d.Ack(false)
			}
		}(i)
	}
}

// =====================================================================
// SIMULATOR PREDICTIVE SCALING
// =====================================================================
func simulatorPredictiveScaling() {
	fmt.Println("📈 Sistem Predictive Scaling aktif memantau jam operasional...")
	for {
		sekarang := time.Now()
		jam := sekarang.Hour()
		tanggal := sekarang.Day()

		if tanggal >= 25 || tanggal <= 3 {
			if jumlahKapasitasServer != 10 {
				jumlahKapasitasServer = 10
				fmt.Printf("\n🔥 [PREDICTIVE SCALING] Tanggal Gajian! Baseline server x%d.\n", jumlahKapasitasServer)
			}
		} else if jam >= 6 && jam <= 18 {
			if jumlahKapasitasServer != 5 {
				jumlahKapasitasServer = 5
				fmt.Printf("\n☀️ [PREDICTIVE SCALING] Jam Sibuk! Baseline server x%d.\n", jumlahKapasitasServer)
			}
		} else {
			if jumlahKapasitasServer != 1 {
				jumlahKapasitasServer = 1
				fmt.Println("\n🌙 [COST OPTIMIZATION] Melakukan Scale-In server.")
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	// Inisialisasi semua komponen
	initDB()
	initRedis()
	initRabbitMQ()
	initCircuitBreaker()

	defer dbMaster.Close()
	defer dbReplica.Close()
	defer mqConn.Close()
	defer mqChannel.Close()

	// Nyalakan fitur Background
	go simulatorPredictiveScaling()
	startRabbitMQWorkers(50) // 50 Goroutine worker (seimbang dengan DB connection pool)

	app := fiber.New()

	// ==========================================
	// SENSOR PROMETHEUS & MIDDLEWARE PROTEKSI
	// ==========================================
	prometheus := fiberprometheus.New("elasticsix_api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(recover.New())              // Anti-Crash Panic
	app.Use(limiter.New(limiter.Config{
		Max:        30, // KEMBALI KETAT! 1 IP / User maksimal 30 request per menit
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { 
			// [TRICK SRE] Baca header IP buatan dari k6. 
			// Jika kosong, baru pakai IP asli laptop.
			simulatedIP := c.Get("X-Simulated-IP")
			if simulatedIP != "" {
				return simulatedIP
			}
			return c.IP() 
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{"status": "error", "message": "Terlalu banyak request."})
		},
	}))
	// ==========================================

	// ==========================================
	// ENDPOINT KESEHATAN SISTEM (HEALTH CHECK)
	// ==========================================
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "success",
			"message": "🚀 API Bank A (ElasticSix) is UP and RUNNING!",
		})
	})

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("PONG! 🏓 Server is healthy!")
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 1: CEK SALDO (Redis Cache + Circuit Breaker + Replica DB)
	// Query ASLI ke PostgreSQL, bukan hardcoded!
	// ------------------------------------------------------------------------
	app.Get("/users/:id/balance", func(c *fiber.Ctx) error {
		noRek := c.Params("id")
		cacheKey := "saldo:" + noRek

		// 1. Cek Redis Cache dulu (fast path)
		cachedBalance, err := rdb.Get(ctx, cacheKey).Result()

		if err == redis.Nil {
			// Cache MISS → Query ke Database Replica via Circuit Breaker
			hasilDB, cbErr := cb.Execute(func() (interface{}, error) {
				var nama string
				var saldo int64
				var jenisRek string
				err := dbReplica.QueryRow(
					"SELECT nama_lengkap, saldo, jenis_rekening FROM nasabah WHERE nomor_rekening = $1 AND status_rekening = 'aktif'",
					noRek,
				).Scan(&nama, &saldo, &jenisRek)
				if err == sql.ErrNoRows {
					// Rekening tidak ditemukan BUKAN error database!
					// Kembalikan nil tanpa error agar Circuit Breaker tidak trip
					return nil, nil
				}
				if err != nil {
					// Error koneksi DB yang sebenarnya → CB harus tahu
					return nil, err
				}
				return fiber.Map{
					"nomor_rekening": noRek,
					"nama":          nama,
					"saldo":         saldo,
					"jenis_rekening": jenisRek,
				}, nil
			})

			if cbErr != nil {
				return c.Status(503).JSON(fiber.Map{
					"status":  "error",
					"source":  "Circuit Breaker",
					"message": "Sistem database sedang gangguan. Coba lagi dalam 15 detik.",
				})
			}

			// Rekening tidak ditemukan (bukan error DB)
			if hasilDB == nil {
				return c.Status(404).JSON(fiber.Map{
					"status":  "error",
					"message": "Rekening tidak ditemukan atau tidak aktif",
				})
			}

			data := hasilDB.(fiber.Map)
			// Simpan saldo ke Redis Cache (TTL 30 detik)
			rdb.Set(ctx, cacheKey, strconv.FormatInt(data["saldo"].(int64), 10), 30*time.Second)

			return c.Status(200).JSON(fiber.Map{
				"status": "success",
				"source": "Database Replica",
				"data":   data,
			})

		} else if err != nil {
			// Redis error → langsung query DB tanpa cache
			var nama string
			var saldo int64
			dbErr := dbReplica.QueryRow(
				"SELECT nama_lengkap, saldo FROM nasabah WHERE nomor_rekening = $1 AND status_rekening = 'aktif'",
				noRek,
			).Scan(&nama, &saldo)
			if dbErr != nil {
				return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Rekening tidak ditemukan"})
			}
			return c.Status(200).JSON(fiber.Map{
				"status": "success",
				"source": "Database Replica (Redis Fallback)",
				"data":   fiber.Map{"nomor_rekening": noRek, "nama": nama, "saldo": saldo},
			})
		}

		// Cache HIT → langsung return dari Redis
		saldo, _ := strconv.ParseInt(cachedBalance, 10, 64)
		return c.Status(200).JSON(fiber.Map{
			"status": "success",
			"source": "Redis Cache",
			"data":   fiber.Map{"nomor_rekening": noRek, "saldo": saldo},
		})
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 2: TRANSFER UANG (Async via RabbitMQ)
	// Request masuk antrean, diproses oleh Worker di background
	// ------------------------------------------------------------------------
	app.Post("/api/v1/transfer", func(c *fiber.Ctx) error {
		var req TransferRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format JSON tidak valid"})
		}

		// Validasi input
		if req.SenderAccount == "" || req.ReceiverAccount == "" || req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{
				"status":  "error",
				"message": "sender_account, receiver_account, dan amount wajib diisi (amount > 0)",
			})
		}

		if req.SenderAccount == req.ReceiverAccount {
			return c.Status(400).JSON(fiber.Map{
				"status":  "error",
				"message": "Tidak bisa transfer ke rekening sendiri",
			})
		}

		body, _ := json.Marshal(req)

		err := mqChannel.PublishWithContext(ctx, "", "transfer_queue", false, false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
			})

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal memasukkan ke antrean server"})
		}

		return c.Status(202).JSON(fiber.Map{
			"status":  "pending",
			"message": "Transaksi diterima dan sedang diproses dalam antrean",
			"data": fiber.Map{
				"sender_account":   req.SenderAccount,
				"receiver_account": req.ReceiverAccount,
				"amount":           req.Amount,
			},
		})
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 3: CEK KESEHATAN DATABASE
	// Menampilkan statistik nasabah & transaksi
	// ------------------------------------------------------------------------
	app.Get("/admin/db-health", func(c *fiber.Ctx) error {
		var totalNasabah, saldoKosong, saldoRendah, nonAktif, totalTransaksi int64
		var minSaldo, maxSaldo, avgSaldo int64

		// Statistik nasabah
		dbReplica.QueryRow(`
			SELECT 
				COUNT(*),
				COUNT(*) FILTER (WHERE saldo <= 0),
				COUNT(*) FILTER (WHERE saldo < 10000),
				COUNT(*) FILTER (WHERE status_rekening != 'aktif'),
				COALESCE(MIN(saldo), 0),
				COALESCE(MAX(saldo), 0),
				COALESCE(ROUND(AVG(saldo)), 0)
			FROM nasabah
		`).Scan(&totalNasabah, &saldoKosong, &saldoRendah, &nonAktif, &minSaldo, &maxSaldo, &avgSaldo)

		// Total transaksi
		dbReplica.QueryRow("SELECT COUNT(*) FROM transaksi").Scan(&totalTransaksi)

		// Status kesehatan
		status := "sehat"
		if saldoKosong > 0 {
			status = "kritis"
		} else if saldoRendah > 100 {
			status = "perlu_refresh"
		}

		return c.JSON(fiber.Map{
			"status":     "success",
			"kesehatan":  status,
			"data": fiber.Map{
				"total_nasabah":       totalNasabah,
				"saldo_kosong":        saldoKosong,
				"saldo_rendah_10rb":   saldoRendah,
				"rekening_non_aktif":  nonAktif,
				"saldo_minimum":       minSaldo,
				"saldo_maksimum":      maxSaldo,
				"saldo_rata_rata":     avgSaldo,
				"total_transaksi":     totalTransaksi,
			},
		})
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 4: REFRESH DATABASE (Reset Saldo & Hapus Riwayat Transaksi)
	// Dipanggil sebelum load test agar data nasabah kembali segar
	// ------------------------------------------------------------------------
	app.Post("/admin/db-refresh", func(c *fiber.Ctx) error {
		// 1. Reset saldo semua nasabah ke nilai random (1jt - 100jt)
		_, err := dbMaster.Exec(`
			UPDATE nasabah 
			SET saldo = 1000000 + (random() * 99000000)::BIGINT,
			    updated_at = NOW()
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal reset saldo: " + err.Error()})
		}

		// 2. Hapus semua riwayat transaksi
		_, err = dbMaster.Exec("DELETE FROM transaksi")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal hapus transaksi: " + err.Error()})
		}

		// 3. Flush Redis cache agar data saldo lama tidak terbaca
		rdb.FlushDB(ctx)

		// 4. Ambil statistik setelah refresh
		var avgSaldo, minSaldo int64
		dbMaster.QueryRow("SELECT ROUND(AVG(saldo)), MIN(saldo) FROM nasabah").Scan(&avgSaldo, &minSaldo)

		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Database berhasil di-refresh! Semua saldo nasabah di-reset dan riwayat transaksi dihapus.",
			"data": fiber.Map{
				"saldo_rata_rata_baru": avgSaldo,
				"saldo_minimum_baru":   minSaldo,
				"cache_redis":          "flushed",
			},
		})
	})

	fmt.Println("🚀 Server API Bank A (ElasticSix) berjalan di Port 8080")
	log.Fatal(app.Listen(":8080"))
}