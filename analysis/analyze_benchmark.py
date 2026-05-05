"""
=============================================================
📊 ANALISIS BENCHMARK: BASELINE vs OPTIMIZED
ElasticSix - Peak Load Management System
=============================================================
Script ini membaca hasil benchmark dari file txt,
menganalisis perbandingan, dan menghasilkan grafik visualisasi.

Jalankan: python analyze_benchmark.py
=============================================================
"""

import re
import os
import matplotlib.pyplot as plt
import matplotlib
import numpy as np

matplotlib.rcParams['font.family'] = 'Segoe UI'
matplotlib.rcParams['font.size'] = 11

# ============================================================
# 1. PARSING HASIL BENCHMARK DARI FILE TXT
# ============================================================
def parse_benchmark(filepath):
    """Membaca file hasil benchmark dan mengekstrak metrik penting."""
    # PowerShell tee menghasilkan UTF-16LE, coba baca dua encoding
    try:
        with open(filepath, 'r', encoding='utf-16') as f:
            content = f.read()
    except (UnicodeError, UnicodeDecodeError):
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()

    metrics = {}

    patterns = {
        'total_requests':  r'Total Requests\s*:\s*([\d,]+)',
        'throughput':      r'Throughput \(RPS\)\s*:\s*([\d.]+)',
        'avg_response':    r'Avg Response Time\s*:\s*([\d.]+)',
        'med_response':    r'Med Response Time\s*:\s*([\d.]+)',
        'p90_response':    r'P90 Response Time\s*:\s*([\d.]+)',
        'p95_response':    r'P95 Response Time\s*:\s*([\d.]+)',
        'max_response':    r'Max Response Time\s*:\s*([\d.]+)',
        'error_rate':      r'Error Rate\s*:\s*([\d.]+)%',
    }

    for key, pattern in patterns.items():
        match = re.search(pattern, content)
        if match:
            val = match.group(1).replace(',', '')
            metrics[key] = float(val)

    # Parse check results
    cek_saldo = re.search(r'Cek Saldo OK.*?(\d+)\s*/\s*.*?(\d+)', content)
    transfer = re.search(r'Transfer OK.*?(\d+)\s*/\s*.*?(\d+)', content)

    if cek_saldo:
        metrics['cek_saldo_pass'] = int(cek_saldo.group(1))
        metrics['cek_saldo_fail'] = int(cek_saldo.group(2))
    if transfer:
        metrics['transfer_pass'] = int(transfer.group(1))
        metrics['transfer_fail'] = int(transfer.group(2))

    return metrics


# ============================================================
# 2. VISUALISASI — GRAFIK PERBANDINGAN
# ============================================================
def create_charts(baseline, optimized, output_dir='.'):
    """Membuat seluruh grafik perbandingan."""

    colors_baseline = '#FF6B6B'   # Merah pastel
    colors_optimized = '#4ECDC4'  # Hijau teal

    fig = plt.figure(figsize=(20, 24))
    fig.suptitle('📊 Analisis Benchmark: Baseline vs Optimized\nElasticSix — Peak Load Management System',
                 fontsize=20, fontweight='bold', y=0.98)

    # -----------------------------------------------------------
    # CHART 1: Response Time Comparison (Grouped Bar)
    # -----------------------------------------------------------
    ax1 = fig.add_subplot(3, 2, 1)

    labels = ['Avg', 'Median', 'P90', 'P95', 'Max']
    base_vals = [baseline['avg_response'], baseline['med_response'],
                 baseline['p90_response'], baseline['p95_response'], baseline['max_response']]
    opt_vals = [optimized['avg_response'], optimized['med_response'],
                optimized['p90_response'], optimized['p95_response'], optimized['max_response']]

    x = np.arange(len(labels))
    width = 0.35

    bars1 = ax1.bar(x - width/2, base_vals, width, label='Baseline', color=colors_baseline, edgecolor='white', linewidth=0.5)
    bars2 = ax1.bar(x + width/2, opt_vals, width, label='Optimized', color=colors_optimized, edgecolor='white', linewidth=0.5)

    ax1.set_ylabel('Response Time (ms)')
    ax1.set_title('⏱️ Response Time Comparison', fontweight='bold', fontsize=14)
    ax1.set_xticks(x)
    ax1.set_xticklabels(labels)
    ax1.legend()
    ax1.grid(axis='y', alpha=0.3)

    # Tambah label nilai di atas bar
    for bar in bars1:
        ax1.text(bar.get_x() + bar.get_width()/2., bar.get_height() + 0.3,
                 f'{bar.get_height():.1f}', ha='center', va='bottom', fontsize=9, fontweight='bold')
    for bar in bars2:
        ax1.text(bar.get_x() + bar.get_width()/2., bar.get_height() + 0.3,
                 f'{bar.get_height():.1f}', ha='center', va='bottom', fontsize=9, fontweight='bold')

    # -----------------------------------------------------------
    # CHART 2: Throughput Comparison (Horizontal Bar)
    # -----------------------------------------------------------
    ax2 = fig.add_subplot(3, 2, 2)

    categories = ['Baseline', 'Optimized']
    throughputs = [baseline['throughput'], optimized['throughput']]

    bars = ax2.barh(categories, throughputs, color=[colors_baseline, colors_optimized],
                    edgecolor='white', height=0.5)

    for bar, val in zip(bars, throughputs):
        ax2.text(bar.get_width() + 5, bar.get_y() + bar.get_height()/2.,
                 f'{val:.1f} req/s', ha='left', va='center', fontweight='bold', fontsize=12)

    ax2.set_xlabel('Requests per Second (RPS)')
    ax2.set_title('🚀 Throughput Comparison', fontweight='bold', fontsize=14)
    ax2.set_xlim(0, max(throughputs) * 1.25)
    ax2.grid(axis='x', alpha=0.3)

    # -----------------------------------------------------------
    # CHART 3: Error Rate Comparison (Bar + Annotation)
    # -----------------------------------------------------------
    ax3 = fig.add_subplot(3, 2, 3)

    error_rates = [baseline['error_rate'], optimized['error_rate']]
    bar_colors = ['#2ecc71' if e < 5 else '#e74c3c' for e in error_rates]

    bars = ax3.bar(categories, error_rates, color=bar_colors, edgecolor='white', width=0.5)

    for bar, val in zip(bars, error_rates):
        ax3.text(bar.get_x() + bar.get_width()/2., bar.get_height() + 0.2,
                 f'{val:.2f}%', ha='center', va='bottom', fontweight='bold', fontsize=14)

    ax3.set_ylabel('Error Rate (%)')
    ax3.set_title('❌ Error Rate Comparison', fontweight='bold', fontsize=14)
    ax3.axhline(y=5, color='orange', linestyle='--', alpha=0.7, label='SLA Threshold (5%)')
    ax3.axhline(y=1, color='green', linestyle='--', alpha=0.7, label='Target (<1%)')
    ax3.legend(fontsize=9)
    ax3.grid(axis='y', alpha=0.3)

    # -----------------------------------------------------------
    # CHART 4: Success/Fail Breakdown (Stacked Bar)
    # -----------------------------------------------------------
    ax4 = fig.add_subplot(3, 2, 4)

    tests = ['Baseline\nCek Saldo', 'Baseline\nTransfer', 'Optimized\nCek Saldo', 'Optimized\nTransfer']
    pass_vals = [
        baseline.get('cek_saldo_pass', 0), baseline.get('transfer_pass', 0),
        optimized.get('cek_saldo_pass', 0), optimized.get('transfer_pass', 0)
    ]
    fail_vals = [
        baseline.get('cek_saldo_fail', 0), baseline.get('transfer_fail', 0),
        optimized.get('cek_saldo_fail', 0), optimized.get('transfer_fail', 0)
    ]

    x = np.arange(len(tests))
    ax4.bar(x, pass_vals, width=0.6, label='✓ Success', color='#2ecc71', edgecolor='white')
    ax4.bar(x, fail_vals, width=0.6, bottom=pass_vals, label='✗ Failed', color='#e74c3c', edgecolor='white')

    ax4.set_xticks(x)
    ax4.set_xticklabels(tests, fontsize=9)
    ax4.set_ylabel('Jumlah Request')
    ax4.set_title('📋 Success vs Failed per Endpoint', fontweight='bold', fontsize=14)
    ax4.legend()
    ax4.grid(axis='y', alpha=0.3)

    # -----------------------------------------------------------
    # CHART 5: Improvement Percentage (Horizontal Bar)
    # -----------------------------------------------------------
    ax5 = fig.add_subplot(3, 2, 5)

    improvement_labels = ['Avg Response', 'Med Response', 'P90 Response', 'P95 Response', 'Max Response', 'Throughput']
    improvements = [
        ((baseline['avg_response'] - optimized['avg_response']) / baseline['avg_response']) * 100,
        ((baseline['med_response'] - optimized['med_response']) / baseline['med_response']) * 100,
        ((baseline['p90_response'] - optimized['p90_response']) / baseline['p90_response']) * 100,
        ((baseline['p95_response'] - optimized['p95_response']) / baseline['p95_response']) * 100,
        ((baseline['max_response'] - optimized['max_response']) / baseline['max_response']) * 100,
        ((optimized['throughput'] - baseline['throughput']) / baseline['throughput']) * 100,
    ]

    bar_colors = ['#2ecc71' if v > 0 else '#e74c3c' for v in improvements]
    bars = ax5.barh(improvement_labels, improvements, color=bar_colors, edgecolor='white', height=0.6)

    for bar, val in zip(bars, improvements):
        offset = 1 if val >= 0 else -1
        ha = 'left' if val >= 0 else 'right'
        ax5.text(bar.get_width() + offset, bar.get_y() + bar.get_height()/2.,
                 f'{val:+.1f}%', ha=ha, va='center', fontweight='bold', fontsize=10)

    ax5.axvline(x=0, color='gray', linewidth=1)
    ax5.set_xlabel('Improvement (%)')
    ax5.set_title('📈 Persentase Peningkatan (Optimized vs Baseline)', fontweight='bold', fontsize=14)
    ax5.grid(axis='x', alpha=0.3)

    # -----------------------------------------------------------
    # CHART 6: Summary Table
    # -----------------------------------------------------------
    ax6 = fig.add_subplot(3, 2, 6)
    ax6.axis('off')

    table_data = [
        ['Total Requests', f"{baseline['total_requests']:,.0f}", f"{optimized['total_requests']:,.0f}",
         f"{((optimized['total_requests']-baseline['total_requests'])/baseline['total_requests'])*100:+.1f}%"],
        ['Throughput (RPS)', f"{baseline['throughput']:.1f}", f"{optimized['throughput']:.1f}",
         f"{((optimized['throughput']-baseline['throughput'])/baseline['throughput'])*100:+.1f}%"],
        ['Avg Response (ms)', f"{baseline['avg_response']:.2f}", f"{optimized['avg_response']:.2f}",
         f"{((baseline['avg_response']-optimized['avg_response'])/baseline['avg_response'])*100:+.1f}%"],
        ['P95 Response (ms)', f"{baseline['p95_response']:.2f}", f"{optimized['p95_response']:.2f}",
         f"{((baseline['p95_response']-optimized['p95_response'])/baseline['p95_response'])*100:+.1f}%"],
        ['Max Response (ms)', f"{baseline['max_response']:.2f}", f"{optimized['max_response']:.2f}",
         f"{((baseline['max_response']-optimized['max_response'])/baseline['max_response'])*100:+.1f}%"],
        ['Error Rate (%)', f"{baseline['error_rate']:.2f}%", f"{optimized['error_rate']:.2f}%",
         f"{optimized['error_rate']-baseline['error_rate']:+.2f}%"],
    ]

    col_labels = ['Metrik', '🐌 Baseline', '🚀 Optimized', 'Δ Perubahan']

    table = ax6.table(
        cellText=table_data,
        colLabels=col_labels,
        cellLoc='center',
        loc='center',
        colWidths=[0.32, 0.22, 0.22, 0.22]
    )

    table.auto_set_font_size(False)
    table.set_fontsize(11)
    table.scale(1, 1.8)

    # Style header
    for j in range(len(col_labels)):
        table[0, j].set_facecolor('#2c3e50')
        table[0, j].set_text_props(color='white', fontweight='bold')

    # Style rows
    for i in range(1, len(table_data) + 1):
        for j in range(len(col_labels)):
            if i % 2 == 0:
                table[i, j].set_facecolor('#f8f9fa')
            else:
                table[i, j].set_facecolor('#ffffff')

    ax6.set_title('📋 Tabel Ringkasan Perbandingan', fontweight='bold', fontsize=14, pad=20)

    plt.tight_layout(rect=[0, 0, 1, 0.96])

    output_path = os.path.join(output_dir, 'benchmark_analysis.png')
    plt.savefig(output_path, dpi=150, bbox_inches='tight', facecolor='white')
    plt.close()
    print(f"✅ Grafik disimpan: {output_path}")
    return output_path


# ============================================================
# 3. CETAK LAPORAN TEKS
# ============================================================
def print_report(baseline, optimized):
    """Cetak analisis teks ke terminal."""

    def pct_change(old, new, lower_is_better=True):
        change = ((old - new) / old) * 100 if lower_is_better else ((new - old) / old) * 100
        return change

    print("\n" + "=" * 65)
    print("  📊 LAPORAN ANALISIS BENCHMARK — ELASTICSIX")
    print("=" * 65)

    print(f"\n{'Metrik':<25} {'Baseline':>12} {'Optimized':>12} {'Δ Change':>12}")
    print("-" * 65)

    metrics_display = [
        ('Total Requests', 'total_requests', False, '{:,.0f}'),
        ('Throughput (RPS)', 'throughput', False, '{:.1f}'),
        ('Avg Response (ms)', 'avg_response', True, '{:.2f}'),
        ('Med Response (ms)', 'med_response', True, '{:.2f}'),
        ('P90 Response (ms)', 'p90_response', True, '{:.2f}'),
        ('P95 Response (ms)', 'p95_response', True, '{:.2f}'),
        ('Max Response (ms)', 'max_response', True, '{:.2f}'),
        ('Error Rate (%)', 'error_rate', True, '{:.2f}'),
    ]

    for label, key, lower_better, fmt in metrics_display:
        bval = baseline[key]
        oval = optimized[key]
        change = pct_change(bval, oval, lower_better)
        icon = "✅" if change > 0 else "⚠️" if change < 0 else "➖"

        print(f"  {label:<23} {fmt.format(bval):>12} {fmt.format(oval):>12} {change:>+8.1f}% {icon}")

    print("-" * 65)

    # Kesimpulan otomatis
    avg_improvement = pct_change(baseline['avg_response'], optimized['avg_response'])
    p95_improvement = pct_change(baseline['p95_response'], optimized['p95_response'])
    throughput_change = pct_change(baseline['throughput'], optimized['throughput'], False)

    print(f"\n  📌 KESIMPULAN OTOMATIS:")
    print(f"  ┌─────────────────────────────────────────────────────────┐")

    if avg_improvement > 0:
        print(f"  │ ✅ Response time LEBIH CEPAT {avg_improvement:.1f}% (avg)              │")
    else:
        print(f"  │ ⚠️  Response time LEBIH LAMBAT {abs(avg_improvement):.1f}% (avg)            │")

    if p95_improvement > 0:
        print(f"  │ ✅ P95 latency TURUN {p95_improvement:.1f}%                            │")

    if throughput_change > 0:
        print(f"  │ ✅ Throughput NAIK {throughput_change:.1f}%                              │")

    if optimized['error_rate'] > baseline['error_rate']:
        print(f"  │ ⚠️  Error rate naik (Rate Limiter aktif memblokir)      │")

    print(f"  └─────────────────────────────────────────────────────────┘")
    print()


# ============================================================
# 4. MAIN
# ============================================================
if __name__ == '__main__':
    baseline_file = 'hasil-baseline.txt'
    optimized_file = 'hasil-optimized.txt'

    # Cek file ada
    if not os.path.exists(baseline_file):
        print(f"❌ File '{baseline_file}' tidak ditemukan!")
        print("   Jalankan dulu: k6 run load-test/benchmark.js 2>&1 | tee hasil-baseline.txt")
        exit(1)

    if not os.path.exists(optimized_file):
        print(f"❌ File '{optimized_file}' tidak ditemukan!")
        print("   Jalankan dulu: k6 run load-test/benchmark.js 2>&1 | tee hasil-optimized.txt")
        exit(1)

    print("📂 Membaca hasil benchmark...")
    baseline = parse_benchmark(baseline_file)
    optimized = parse_benchmark(optimized_file)

    print(f"   Baseline:  {len(baseline)} metrik terbaca")
    print(f"   Optimized: {len(optimized)} metrik terbaca")

    # Cetak laporan teks
    print_report(baseline, optimized)

    # Buat grafik
    print("🎨 Membuat grafik perbandingan...")
    try:
        chart_path = create_charts(baseline, optimized)
        print(f"\n🎉 Selesai! Buka file '{chart_path}' untuk melihat grafik.")
    except Exception as e:
        print(f"⚠️  Gagal membuat grafik: {e}")
        print("   Pastikan matplotlib terinstall: pip install matplotlib")
