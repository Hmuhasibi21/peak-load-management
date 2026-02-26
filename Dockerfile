# ==========================================
# STAGE 1: BUILDER (Fase Kompilasi Go)
# ==========================================
FROM golang:1.21-alpine AS builder

# Set direktori kerja di dalam container
WORKDIR /app

# Copy seluruh source code ke dalam container
COPY . .

# Download dependency (jika nanti Fathur sudah buat go.mod)
RUN go mod download || true

# Compile aplikasi Go menjadi file binary bernama 'api-server'
# Mengambil source code dari cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o api-server ./cmd/api/main.go

# ==========================================
# STAGE 2: RUNNER (Fase Produksi - Sangat Ringan)
# ==========================================
FROM alpine:latest

WORKDIR /root/

# Copy HANYA file binary hasil kompilasi dari Stage 1
COPY --from=builder /app/api-server .

# Buka port 8080 agar k6 dan Postman bisa menembak ke sini
EXPOSE 8080

# Jalankan aplikasi saat container menyala
CMD ["./api-server"]