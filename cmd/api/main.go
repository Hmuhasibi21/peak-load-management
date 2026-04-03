package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ansrivas/fiberprometheus/v2" // Tambahan library sensor Prometheus
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

var db *sql.DB
var rdb *redis.Client
var mqConn *amqp.Connection
var mqChannel *amqp.Channel
var ctx = context.Background()

// 1. Init Database (Single DB untuk POC)
func initDB() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Disesuaikan dengan docker-compose.yml milik ElasticSix
		dbURL = "postgres://root:password@postgres:5432/bank_a_db?sslmode=disable"
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Gagal konek ke Database:", err)
	}
	fmt.Println("✅ Berhasil terhubung ke PostgreSQL Database (Single DB)!")
}

// 2. Init Redis Cache
func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // <--- Koma ganda yang memicu error sudah dihapus
		Password: "",
		DB:       0,
	})
	fmt.Println("✅ Redis Client disiapkan!")
}

// 3. Init RabbitMQ (Untuk Antrean Transfer)
func initRabbitMQ() {
	var err error
	// Disesuaikan dengan kredensial docker-compose.yml
	mqConn, err = amqp.Dial("amqp://admin:admin123@rabbitmq:5672/")
	if err != nil {
		log.Fatal("Gagal konek ke RabbitMQ:", err)
	}

	mqChannel, err = mqConn.Channel()
	if err != nil {
		log.Fatal("Gagal buka channel RabbitMQ:", err)
	}

	// Deklarasi Antrean (Queue)
	_, err = mqChannel.QueueDeclare(
		"transfer_queue", // nama antrean
		true,             // durable (Aman! Jika server mati, antrean tidak hilang)
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Fatal("Gagal deklarasi queue RabbitMQ:", err)
	}
	fmt.Println("✅ RabbitMQ siap menerima antrean transfer!")
}

// Struct untuk menerima request JSON
type TransferRequest struct {
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Amount     float64 `json:"amount"`
}

func main() {
	initDB()
	initRedis()
	initRabbitMQ()
	
	// Pastikan koneksi ditutup saat aplikasi mati
	defer db.Close()
	defer mqConn.Close()
	defer mqChannel.Close()

	app := fiber.New()

	// ==========================================
	// 🚨 SENSOR PROMETHEUS DIMULAI DI SINI
	// ==========================================
	// Membuat sensor khusus untuk service ini
	prometheus := fiberprometheus.New("elasticsix_api") 
	
	// Membuat rute http://localhost:8080/metrics agar bisa disedot Prometheus
	prometheus.RegisterAt(app, "/metrics")              
	
	// Aktifkan sensor untuk memantau SEMUA endpoint (Kecepatan, Error, Jumlah Request)
	app.Use(prometheus.Middleware)                      
	// ==========================================

	// ------------------------------------------------------------------------
	// ENDPOINT 1: Cek Saldo (BALANCE INQUIRY) - Pake REDIS
	// ------------------------------------------------------------------------
	app.Get("/users/:id/balance", func(c *fiber.Ctx) error {
		userID := c.Params("id")
		cacheKey := "balance_user_" + userID

		cachedBalance, err := rdb.Get(ctx, cacheKey).Result()
		if err == redis.Nil {
			fmt.Println("⚠️ Cache Miss! Ambil dari PostgreSQL untuk User:", userID)
			
			// Nanti tim TIF akan mengganti ini dengan query "SELECT balance FROM accounts..."
			saldoDariDB := "Rp 5.000.000" 

			// Simpan ke Redis (TTL 30 detik)
			rdb.Set(ctx, cacheKey, saldoDariDB, 30*time.Second)

			return c.Status(200).JSON(fiber.Map{
				"status":  "success",
				"source":  "Database PostgreSQL (Lambat)",
				"balance": saldoDariDB,
			})
		} else if err != nil {
			return c.Status(500).SendString("Kesalahan Server Internal")
		}

		fmt.Println("⚡ Cache Hit! Ambil dari Redis untuk User:", userID)
		return c.Status(200).JSON(fiber.Map{
			"status":  "success",
			"source":  "Redis Cache (Sangat Cepat!)",
			"balance": cachedBalance,
		})
	})

	// ------------------------------------------------------------------------
	// ENDPOINT 2: TRANSFER UANG - Dilempar ke RABBITMQ
	// ------------------------------------------------------------------------
	app.Post("/api/v1/transfer", func(c *fiber.Ctx) error {
		var req TransferRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Format JSON tidak valid"})
		}

		// Ubah request JSON tadi menjadi bentuk Bytes untuk dimasukkan ke RabbitMQ
		body, _ := json.Marshal(req)

		// Push ke Antrean RabbitMQ
		err := mqChannel.PublishWithContext(ctx,
			"",               // exchange
			"transfer_queue", // routing key (nama queue)
			false,            // mandatory
			false,            // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent, // Wajib Persistent agar uang aman di disk!
				Body:         body,
			})

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal memasukkan ke antrean server"})
		}

		// Langsung beri respon ke User meskipun belum masuk ke Database (Asynchronous)
		return c.Status(202).JSON(fiber.Map{
			"status":  "pending",
			"message": "Transaksi diterima dan sedang diproses dalam antrean",
		})
	})

	// Menjalankan Server di Port 8080 (Sesuai tes lokalmu agar tidak bentrok)
	fmt.Println("🚀 Server API Bank A (ElasticSix) berjalan di Port 8080")
	log.Fatal(app.Listen(":8080"))
}