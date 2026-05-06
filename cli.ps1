# =====================================================
# ElasticSix - Peak Load Management CLI
# =====================================================
# Cara pakai:
#   1. Buka PowerShell di folder project
#   2. Ketik: . .\cli.ps1
#   3. Ketik: elk
# =====================================================

function elk {
    Write-Host ''
    Write-Host '  =========================================' -ForegroundColor Cyan
    Write-Host '   ElasticSix CLI - Peak Load Management' -ForegroundColor Cyan
    Write-Host '  =========================================' -ForegroundColor Cyan
    Write-Host ''
    Write-Host '  BUILD' -ForegroundColor Yellow
    Write-Host '    elk-up              Jalankan stack Optimized'
    Write-Host '    elk-up-baseline     Jalankan stack Baseline'
    Write-Host '    elk-down            Matikan Optimized + hapus volume'
    Write-Host '    elk-down-baseline   Matikan Baseline + hapus volume'
    Write-Host '    elk-restart         Rebuild Optimized dari awal'
    Write-Host '    elk-logs            Lihat log API instance 1'
    Write-Host '    elk-ps              Lihat status container'
    Write-Host ''
    Write-Host '  DATABASE' -ForegroundColor Yellow
    Write-Host '    elk-db-health       Cek kesehatan database'
    Write-Host '    elk-db-refresh      Reset saldo + hapus transaksi'
    Write-Host '    elk-db-sql          Masuk ke psql Master'
    Write-Host ''
    Write-Host '  LOAD TEST' -ForegroundColor Yellow
    Write-Host '    elk-peak            Peak load test biasa'
    Write-Host '    elk-bench           Benchmark Optimized random IP'
    Write-Host '    elk-bench-real      Benchmark Optimized IP unik'
    Write-Host '    elk-bench-baseline  Benchmark Baseline'
    Write-Host '    elk-bench-all       Full auto: Baseline + Optimized + Analisis'
    Write-Host ''
    Write-Host '  ANALISIS' -ForegroundColor Yellow
    Write-Host '    elk-analyze         Python analisis + grafik'
    Write-Host '    elk-ping            Cek load balancing'
    Write-Host ''
    Write-Host '  MONITORING' -ForegroundColor Yellow
    Write-Host '    elk-grafana         Buka Grafana di browser'
    Write-Host '    elk-rabbit          Buka RabbitMQ di browser'
    Write-Host ''
}

# =====================================================
# BUILD
# =====================================================
function elk-up {
    Write-Host '[elk] Building Optimized stack...' -ForegroundColor Green
    docker compose up --build -d
    Write-Host '[elk] Waiting 30s...' -ForegroundColor Yellow
    Start-Sleep -Seconds 30
    curl.exe -s http://localhost/ping
    Write-Host ''
    Write-Host '[elk] Ready!' -ForegroundColor Green
}

function elk-up-baseline {
    Write-Host '[elk] Building Baseline stack...' -ForegroundColor Green
    docker compose -f docker-compose.baseline.yml up --build -d
    Write-Host '[elk] Waiting 30s...' -ForegroundColor Yellow
    Start-Sleep -Seconds 30
    curl.exe -s http://localhost/ping
    Write-Host ''
    Write-Host '[elk] Ready!' -ForegroundColor Green
}

function elk-down {
    Write-Host '[elk] Stopping Optimized...' -ForegroundColor Red
    docker compose down -v
}

function elk-down-baseline {
    Write-Host '[elk] Stopping Baseline...' -ForegroundColor Red
    docker compose -f docker-compose.baseline.yml down -v
}

function elk-restart {
    Write-Host '[elk] Full restart Optimized...' -ForegroundColor Yellow
    docker compose down -v
    docker compose up --build -d
    Write-Host '[elk] Waiting 30s...' -ForegroundColor Yellow
    Start-Sleep -Seconds 30
    curl.exe -s http://localhost/ping
    Write-Host ''
    Write-Host '[elk] Restarted!' -ForegroundColor Green
}

function elk-logs {
    docker logs elasticsix_api_1 --tail 50
}

function elk-ps {
    docker compose ps
}

# =====================================================
# DATABASE
# =====================================================
function elk-db-health {
    Write-Host '[elk] Checking database...' -ForegroundColor Cyan
    curl.exe -s http://localhost/admin/db-health | python -m json.tool
}

function elk-db-refresh {
    Write-Host '[elk] Refreshing database...' -ForegroundColor Yellow
    curl.exe -s -X POST http://localhost/admin/db-refresh | python -m json.tool
}

function elk-db-sql {
    docker exec -it elasticsix_postgres_master psql -U root -d bank_a_db
}

# =====================================================
# LOAD TEST & BENCHMARK
# =====================================================
function elk-peak {
    Write-Host '[elk] Running peak load test...' -ForegroundColor Cyan
    k6 run load-test/peak-test.js
}

function elk-bench {
    Write-Host '[elk] Benchmark Optimized...' -ForegroundColor Cyan
    k6 run load-test/benchmark.js 2>&1 | Tee-Object -FilePath results/hasil-optimized.txt
}

function elk-bench-real {
    Write-Host '[elk] Benchmark Optimized real-life...' -ForegroundColor Cyan
    k6 run load-test/benchmark-reallife.js 2>&1 | Tee-Object -FilePath results/hasil-optimized-reallife.txt
}

function elk-bench-baseline {
    Write-Host '[elk] Benchmark Baseline...' -ForegroundColor Cyan
    k6 run load-test/benchmark.js 2>&1 | Tee-Object -FilePath results/hasil-baseline.txt
}

function elk-bench-all {
    Write-Host '================================================' -ForegroundColor Cyan
    Write-Host '  FULL BENCHMARK: Baseline - Optimized - Analisis' -ForegroundColor Cyan
    Write-Host '================================================' -ForegroundColor Cyan

    Write-Host '[1/5] Starting Baseline...' -ForegroundColor Yellow
    docker compose down -v 2>$null
    docker compose -f docker-compose.baseline.yml up --build -d
    Write-Host '[1/5] Waiting 30s...' -ForegroundColor Yellow
    Start-Sleep -Seconds 30

    Write-Host '[2/5] Benchmark Baseline...' -ForegroundColor Yellow
    k6 run load-test/benchmark.js 2>&1 | Tee-Object -FilePath results/hasil-baseline.txt

    Write-Host '[3/5] Switching to Optimized...' -ForegroundColor Yellow
    docker compose -f docker-compose.baseline.yml down -v
    docker compose up --build -d
    Write-Host '[3/5] Waiting 30s...' -ForegroundColor Yellow
    Start-Sleep -Seconds 30

    Write-Host '[4/5] Benchmark Optimized...' -ForegroundColor Yellow
    k6 run load-test/benchmark.js 2>&1 | Tee-Object -FilePath results/hasil-optimized.txt

    Write-Host '[5/5] Analyzing...' -ForegroundColor Yellow
    $env:PYTHONIOENCODING = 'utf-8'
    python analysis/analyze_benchmark.py

    Write-Host '' 
    Write-Host '================================================' -ForegroundColor Green
    Write-Host '  DONE! Output files:' -ForegroundColor Green
    Write-Host '    results/hasil-baseline.txt'
    Write-Host '    results/hasil-optimized.txt'
    Write-Host '    results/benchmark_analysis.png'
    Write-Host '================================================' -ForegroundColor Green
}

# =====================================================
# ANALISIS
# =====================================================
function elk-analyze {
    Write-Host '[elk] Running analysis...' -ForegroundColor Cyan
    $env:PYTHONIOENCODING = 'utf-8'
    python analysis/analyze_benchmark.py
}

function elk-ping {
    Write-Host '[elk] Pinging API 3x...' -ForegroundColor Cyan
    1..3 | ForEach-Object {
        $r = curl.exe -sI http://localhost/ping 2>$null | Select-String 'X-Upstream'
        Write-Host "  Request $_ -> $r"
    }
    Write-Host ''
}

# =====================================================
# MONITORING
# =====================================================
function elk-grafana {
    Start-Process 'http://localhost:3000'
    Write-Host '[elk] Opening Grafana...' -ForegroundColor Cyan
}

function elk-rabbit {
    Start-Process 'http://localhost:15672'
    Write-Host '[elk] Opening RabbitMQ...' -ForegroundColor Cyan
}

# =====================================================
Write-Host ''
Write-Host '  ElasticSix CLI loaded! Ketik elk untuk lihat semua command.' -ForegroundColor Green
Write-Host ''
