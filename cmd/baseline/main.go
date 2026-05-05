package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

// =====================================================================
// BASELINE API — TANPA OPTIMASI APAPUN
// Tujuan: Menjadi pembanding untuk mengukur efektivitas fitur SRE
// =====================================================================

var db *sql.DB

type TransferRequest struct {
	SenderAccount   string `json:"sender_account"`
	ReceiverAccount string `json:"receiver_account"`
	Amount          int64  `json:"amount"`
	Keterangan      string `json:"keterangan"`
}

func initDB() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:password@localhost:5432/bank_a_db?sslmode=disable"
	}

	// Retry koneksi (DB mungkin belum siap)
	for i := 1; i <= 15; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				break
			}
		}
		fmt.Printf("⏳ Menunggu Database siap... (Percobaan %d/15)\n", i)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("❌ Gagal konek ke Database:", err)
	}

	// Connection pool standar (tidak dioptimasi)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	fmt.Println("🗄️ Database terhubung (Baseline — tanpa Read/Write Split)")
}

func main() {
	initDB()
	defer db.Close()

	app := fiber.New()

	// Prometheus (tetap ada untuk monitoring perbandingan)
	prometheus := fiberprometheus.New("elasticsix_baseline")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)
	app.Use(recover.New())

	// TANPA Rate Limiter!

	// Health Check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "message": "Baseline API (Tanpa Optimasi)"})
	})

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("PONG! 🏓 Baseline Server")
	})

	// ================================================================
	// CEK SALDO — LANGSUNG QUERY KE DATABASE (TANPA CACHE, TANPA CB)
	// ================================================================
	app.Get("/users/:id/balance", func(c *fiber.Ctx) error {
		noRek := c.Params("id")

		var nama string
		var saldo int64
		var jenisRek string

		// Query langsung ke database, setiap request = 1 query
		err := db.QueryRow(
			"SELECT nama_lengkap, saldo, jenis_rekening FROM nasabah WHERE nomor_rekening = $1 AND status_rekening = 'aktif'",
			noRek,
		).Scan(&nama, &saldo, &jenisRek)

		if err != nil {
			return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Rekening tidak ditemukan"})
		}

		return c.Status(200).JSON(fiber.Map{
			"status": "success",
			"source": "Database (Direct Query)",
			"data": fiber.Map{
				"nomor_rekening":  noRek,
				"nama":           nama,
				"saldo":          saldo,
				"jenis_rekening": jenisRek,
			},
		})
	})

	// ================================================================
	// TRANSFER — SYNCHRONOUS (LANGSUNG INSERT, TANPA QUEUE)
	// ================================================================
	app.Post("/api/v1/transfer", func(c *fiber.Ctx) error {
		var req TransferRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format JSON tidak valid"})
		}

		if req.SenderAccount == "" || req.ReceiverAccount == "" || req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Data tidak lengkap"})
		}

		// SYNCHRONOUS: Semua diproses langsung di dalam request handler
		tx, err := db.Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal mulai transaksi"})
		}

		// 1. Cek saldo pengirim (dengan lock)
		var saldoPengirim int64
		err = tx.QueryRow(
			"SELECT saldo FROM nasabah WHERE nomor_rekening = $1 AND status_rekening = 'aktif' FOR UPDATE",
			req.SenderAccount,
		).Scan(&saldoPengirim)
		if err != nil {
			tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Rekening pengirim tidak ditemukan"})
		}

		if saldoPengirim < req.Amount {
			tx.Rollback()
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Saldo tidak cukup"})
		}

		// 2. Kurangi saldo pengirim
		_, err = tx.Exec("UPDATE nasabah SET saldo = saldo - $1, updated_at = NOW() WHERE nomor_rekening = $2",
			req.Amount, req.SenderAccount)
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal update saldo pengirim"})
		}

		// 3. Tambah saldo penerima
		_, err = tx.Exec("UPDATE nasabah SET saldo = saldo + $1, updated_at = NOW() WHERE nomor_rekening = $2",
			req.Amount, req.ReceiverAccount)
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal update saldo penerima"})
		}

		// 4. Insert transaksi
		noRef := fmt.Sprintf("BL%d", time.Now().UnixNano())
		keterangan := req.Keterangan
		if keterangan == "" {
			keterangan = "Transfer baseline"
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
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal catat transaksi"})
		}

		// 5. Commit
		if err = tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal commit transaksi"})
		}

		// Response SYNCHRONOUS: 200 OK (bukan 202 Accepted)
		return c.Status(200).JSON(fiber.Map{
			"status":  "success",
			"message": "Transfer berhasil diproses (synchronous)",
			"data":    fiber.Map{"sender": req.SenderAccount, "receiver": req.ReceiverAccount, "amount": req.Amount},
		})
	})

	fmt.Println("🐌 Baseline API (Tanpa Optimasi) berjalan di Port 8080")
	log.Fatal(app.Listen(":8080"))
}

// Ensure json import is used
var _ = json.Marshal
