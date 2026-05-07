#!/bin/bash

# =====================================================
# ElasticSix - Peak Load Management CLI (macOS/Linux)
# =====================================================
# Cara pakai:
#   1. Buka Terminal di folder project
#   2. Beri izin eksekusi sekali: chmod +x cli.sh
#   3. Jalankan command: ./cli.sh <command>
#      Contoh: ./cli.sh elk-up
# =====================================================

function elk_help {
    echo ""
    echo -e "\033[36m  =========================================\033[0m"
    echo -e "\033[36m   ElasticSix CLI - Peak Load Management\033[0m"
    echo -e "\033[36m  =========================================\033[0m"
    echo ""
    echo -e "\033[33m  BUILD\033[0m"
    echo "    elk-up              Jalankan stack Optimized"
    echo "    elk-up-baseline     Jalankan stack Baseline"
    echo "    elk-down            Matikan Optimized + hapus volume"
    echo "    elk-down-baseline   Matikan Baseline + hapus volume"
    echo "    elk-restart         Rebuild Optimized dari awal"
    echo "    elk-logs            Lihat log API instance 1"
    echo "    elk-ps              Lihat status container"
    echo ""
    echo -e "\033[33m  DATABASE\033[0m"
    echo "    elk-db-health       Cek kesehatan database"
    echo "    elk-db-refresh      Reset saldo + hapus transaksi"
    echo "    elk-db-sql          Masuk ke psql Master"
    echo ""
    echo -e "\033[33m  LOAD TEST\033[0m"
    echo "    elk-peak            Peak load test biasa"
    echo "    elk-bench           Benchmark Optimized random IP"
    echo "    elk-bench-real      Benchmark Optimized IP unik"
    echo "    elk-bench-baseline  Benchmark Baseline"
    echo "    elk-bench-all       Full auto: Baseline + Optimized + Analisis"
    echo ""
    echo -e "\033[33m  ANALISIS\033[0m"
    echo "    elk-analyze         Python analisis + grafik"
    echo "    elk-ping            Cek load balancing"
    echo ""
    echo -e "\033[33m  MONITORING\033[0m"
    echo "    elk-grafana         Buka Grafana di browser"
    echo "    elk-rabbit          Buka RabbitMQ di browser"
    echo ""
}

# =====================================================
# BUILD
# =====================================================
function elk-up {
    echo -e "\033[32m[elk] Building Optimized stack...\033[0m"
    docker compose up --build -d
    echo -e "\033[33m[elk] Waiting 30s...\033[0m"
    sleep 30
    curl -s http://localhost/ping
    echo ""
    echo -e "\033[32m[elk] Ready!\033[0m"
}

function elk-up-baseline {
    echo -e "\033[32m[elk] Building Baseline stack...\033[0m"
    docker compose -f docker-compose.baseline.yml up --build -d
    echo -e "\033[33m[elk] Waiting 30s...\033[0m"
    sleep 30
    curl -s http://localhost/ping
    echo ""
    echo -e "\033[32m[elk] Ready!\033[0m"
}

function elk-down {
    echo -e "\033[31m[elk] Stopping Optimized...\033[0m"
    docker compose down -v
}

function elk-down-baseline {
    echo -e "\033[31m[elk] Stopping Baseline...\033[0m"
    docker compose -f docker-compose.baseline.yml down -v
}

function elk-restart {
    echo -e "\033[33m[elk] Full restart Optimized...\033[0m"
    docker compose down -v
    docker compose up --build -d
    echo -e "\033[33m[elk] Waiting 30s...\033[0m"
    sleep 30
    curl -s http://localhost/ping
    echo ""
    echo -e "\033[32m[elk] Restarted!\033[0m"
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
    echo -e "\033[36m[elk] Checking database...\033[0m"
    curl -s http://localhost/admin/db-health | python3 -m json.tool
}

function elk-db-refresh {
    echo -e "\033[33m[elk] Refreshing database...\033[0m"
    curl -s -X POST http://localhost/admin/db-refresh | python3 -m json.tool
}

function elk-db-sql {
    docker exec -it elasticsix_postgres_master psql -U root -d bank_a_db
}

# =====================================================
# LOAD TEST & BENCHMARK
# =====================================================
function elk-peak {
    echo -e "\033[36m[elk] Running peak load test...\033[0m"
    k6 run load-test/peak-test.js
}

function elk-bench {
    echo -e "\033[36m[elk] Benchmark Optimized...\033[0m"
    k6 run load-test/benchmark.js 2>&1 | tee results/hasil-optimized.txt
}

function elk-bench-real {
    echo -e "\033[36m[elk] Benchmark Optimized real-life...\033[0m"
    k6 run load-test/benchmark-reallife.js 2>&1 | tee results/hasil-optimized-reallife.txt
}

function elk-bench-baseline {
    echo -e "\033[36m[elk] Benchmark Baseline...\033[0m"
    k6 run load-test/benchmark.js 2>&1 | tee results/hasil-baseline.txt
}

function elk-bench-all {
    echo -e "\033[36m================================================\033[0m"
    echo -e "\033[36m  FULL BENCHMARK: Baseline - Optimized - Analisis\033[0m"
    echo -e "\033[36m================================================\033[0m"

    echo -e "\033[33m[1/5] Starting Baseline...\033[0m"
    docker compose down -v 2>/dev/null
    docker compose -f docker-compose.baseline.yml up --build -d
    echo -e "\033[33m[1/5] Waiting 30s...\033[0m"
    sleep 30

    echo -e "\033[33m[2/5] Benchmark Baseline...\033[0m"
    k6 run load-test/benchmark.js 2>&1 | tee results/hasil-baseline.txt

    echo -e "\033[33m[3/5] Switching to Optimized...\033[0m"
    docker compose -f docker-compose.baseline.yml down -v
    docker compose up --build -d
    echo -e "\033[33m[3/5] Waiting 30s...\033[0m"
    sleep 30

    echo -e "\033[33m[4/5] Benchmark Optimized...\033[0m"
    k6 run load-test/benchmark.js 2>&1 | tee results/hasil-optimized.txt

    echo -e "\033[33m[5/5] Analyzing...\033[0m"
    export PYTHONIOENCODING=utf-8
    python3 analysis/analyze_benchmark.py

    echo "" 
    echo -e "\033[32m================================================\033[0m"
    echo -e "\033[32m  DONE! Output files:\033[0m"
    echo -e "\033[32m    results/hasil-baseline.txt\033[0m"
    echo -e "\033[32m    results/hasil-optimized.txt\033[0m"
    echo -e "\033[32m    results/benchmark_analysis_<waktu>.png\033[0m"
    echo -e "\033[32m================================================\033[0m"
}

# =====================================================
# ANALISIS
# =====================================================
function elk-analyze {
    echo -e "\033[36m[elk] Running analysis...\033[0m"
    export PYTHONIOENCODING=utf-8
    python3 analysis/analyze_benchmark.py
}

function elk-ping {
    echo -e "\033[36m[elk] Pinging API 3x...\033[0m"
    for i in {1..3}; do
        r=$(curl -sI http://localhost/ping 2>/dev/null | grep -i 'X-Upstream')
        echo "  Request $i -> $r"
    done
    echo ""
}

# =====================================================
# MONITORING
# =====================================================
function elk-grafana {
    open 'http://localhost:3000'
    echo -e "\033[36m[elk] Opening Grafana...\033[0m"
}

function elk-rabbit {
    open 'http://localhost:15672'
    echo -e "\033[36m[elk] Opening RabbitMQ...\033[0m"
}

# =====================================================
# MAIN ROUTING
# =====================================================
# Menentukan apakah script dijalankan langsung (bukan di-source)
is_sourced=false
if [[ -n "$ZSH_VERSION" ]]; then
    # Zsh: Jika kita sedang men-source file, zsh_eval_context akan mengandung 'file'
    if [[ "$ZSH_EVAL_CONTEXT" =~ "toplevel" || "$ZSH_EVAL_CONTEXT" =~ "file" ]]; then
        # Jika dijalankan via ./cli.sh, konteksnya hanya 'toplevel'. Jika di-source, konteksnya 'toplevel:file' atau 'toplevel:source' dll.
        # Cara termudah di Zsh adalah mengecek apakah $0 mengandung nama script ini.
        if [[ ! "$0" =~ "cli.sh" ]]; then
            is_sourced=true
        fi
    fi
elif [[ -n "$BASH_VERSION" ]]; then
    if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
        is_sourced=true
    fi
fi

if [[ "$is_sourced" == false ]]; then
    if [[ -z "$1" ]]; then
        elk_help
    elif declare -f "$1" > /dev/null; then
        "$@"
    else
        echo -e "\033[31mError: Command '$1' not found.\033[0m"
        elk_help
        exit 1
    fi
fi
