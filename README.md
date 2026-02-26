# 🚀 Peak Load Management System
**By Team ElasticSix (Capstone Project FILKOM UB 2026)**

Repository ini berisi prototipe arsitektur sistem backend yang dirancang khusus untuk menangani *Exploding User Data* dan kondisi *Peak Load* ekstrem (simulasi 1 juta transaksi/jam). 

Sistem ini dioptimasi menggunakan **Golang (Fiber)** dan mengimplementasikan mekanisme ketahanan sistem (SRE) seperti:
- **Rate Limiting & Circuit Breaker** (API Protection)
- **Redis Caching** (Read/Write Separation)
- **RabbitMQ Async Processing** (Backpressure/Queueing)
- **Observability** (Prometheus & Grafana)

Dideploy sepenuhnya menggunakan Docker Compose untuk pengujian *Load Testing* (k6) lokal secara efisien.