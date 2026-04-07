package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

// Struct JSON Transfer
type TransferRequest struct {
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Amount     float64 `json:"amount"`
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

	dbReplica, err = sql.Open("postgres", replicaURL)
	if err != nil {
		log.Fatal("Gagal konek ke Replica DB:", err)
	}

	fmt.Println("🗄️ Berhasil terhubung ke Master dan Replica Database!")
}

// 2. Init Redis Cache
func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	fmt.Println("⚡ Redis Client disiapkan!")
}

// 3. Init RabbitMQ
func initRabbitMQ() {
	var err error
	mqConn, err = amqp.Dial("amqp://admin:admin123@localhost:5672/")
	if err != nil {
		log.Fatal("Gagal konek ke RabbitMQ:", err)
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
// BACKGROUND WORKER (RABBITMQ CONSUMER)
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
				// Simulasi waktu insert data ke dbMaster
				time.Sleep(1 * time.Second)

				// Tandai bahwa pesan sudah sukses diproses (Manual Ack)
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
	startRabbitMQWorkers(100) // Nyalakan 100 Goroutine pembaca RabbitMQ

	app := fiber.New()

	// ==========================================
	// SENSOR PROMETHEUS & MIDDLEWARE PROTEKSI
	// ==========================================
	prometheus := fiberprometheus.New("elasticsix_api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(recover.New())              // Anti-Crash Panic
	app.Use(limiter.New(limiter.Config{ // Rate Limiting
		Max:          20,
		Expiration:   1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
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
	// ENDPOINT 1: BALANCE INQUIRY (Redis + Circuit Breaker + Replica DB)
	// ------------------------------------------------------------------------
	app.Get("/users/:id/balance", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		cacheKey := "balance_user_" + userID

		cachedBalance, err := rdb.Get(ctx, cacheKey).Result()

		if err == redis.Nil { // Cache Miss
			hasilDB, cbErr := cb.Execute(func() (interface{}, error) {
				// Simulasi error untuk demo Circuit Breaker ke dosen
				if userID == "error" {
					return nil, fmt.Errorf("database timeout")
				}
				return "Rp 5.000.000", nil // Simulasi ambil dari dbReplica
			})

			if cbErr != nil {
				return c.Status(503).JSON(fiber.Map{
					"status":  "error",
					"source":  "Circuit Breaker",
					"message": "Sistem database gangguan. Coba lagi 15 detik.",
				})
			}

			saldoDariDB := hasilDB.(string)
			rdb.Set(ctx, cacheKey, saldoDariDB, 30*time.Second)
			return c.Status(200).JSON(fiber.Map{"source": "Database Replica", "balance": saldoDariDB})

		} else if err != nil {
			return c.Status(200).JSON(fiber.Map{"source": "Database Replica (Fallback)", "balance": "Rp 5.000.000"})
		}

		return c.Status(200).JSON(fiber.Map{"source": "Redis Cache", "balance": cachedBalance})
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 2: TRANSFER UANG (RabbitMQ)
	// ------------------------------------------------------------------------
	app.Post("/api/v1/transfer", func(c *fiber.Ctx) error {
		var req TransferRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Format JSON tidak valid"})
		}

		body, _ := json.Marshal(req)

		err := mqChannel.PublishWithContext(ctx, "", "transfer_queue", false, false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
			})

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal memasukkan ke antrean server"})
		}

		return c.Status(202).JSON(fiber.Map{
			"status":  "pending",
			"message": "Transaksi diterima dan sedang diproses dalam antrean",
		})
	})

	fmt.Println("🚀 Server API Bank A (ElasticSix) berjalan di Port 8080")
	log.Fatal(app.Listen(":8080"))
}
