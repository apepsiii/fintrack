#!/bin/bash
set -e

# ============================================================
#  FinTrack - Build Script
#
#  Builds linux binaries (CGO_ENABLED=0, stripped) into ./dist/
#  and prints upload instructions.
#
#  After upload, run setup.sh on the VPS for the wizard menu
#  (install / update / status).
#
#  Usage:
#    sh build.sh              # build amd64 + arm64
#    sh build.sh amd64        # amd64 only
#    sh build.sh arm64        # arm64 only (Raspberry Pi, ARM VPS)
# ============================================================

VERSION="${VERSION:-v1.0.0}"
LDFLAGS="-s -w"
DIST="dist"
APP_NAME="fintrack"

ARCH_ARG="${1:-both}"
case "$ARCH_ARG" in
    amd64) TARGETS=("amd64") ;;
    arm64) TARGETS=("arm64") ;;
    both)  TARGETS=("amd64" "arm64") ;;
    *) echo "Usage: sh build.sh [amd64|arm64|both]"; exit 1 ;;
esac

echo ""
echo "========================================"
echo "  FinTrack - Build ${VERSION}"
echo "  $(date +%Y-%m-%d)"
echo "========================================"

rm -rf "$DIST"
mkdir -p "$DIST"

BUILT=()
for goarch in "${TARGETS[@]}"; do
    OUT="$DIST/${APP_NAME}_linux_${goarch}"
    echo ""
    echo "[*] Building linux/${goarch} (CGO_ENABLED=0, stripped)..."
    GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 \
        go build -ldflags="${LDFLAGS}" -o "${OUT}" .
    SIZE=$(du -h "${OUT}" | cut -f1)
    echo "    -> ${SIZE}  ${OUT}"
    BUILT+=("$goarch")
done

# Copy setup wizard dan assets yang diperlukan
cp setup.sh "$DIST/setup.sh" 2>/dev/null && echo "    setup.sh (wizard installer)"

# Copy templates dan static files
cp -r templates "$DIST/templates" 2>/dev/null && echo "    templates/ (HTML)"
cp -r static "$DIST/static" 2>/dev/null && echo "    static/ (assets)"

# Copy .env.example sebagai referensi
cp .env.example "$DIST/.env.example" 2>/dev/null && echo "    .env.example (konfigurasi)"

echo ""
echo "========================================"
echo "  BUILD SELESAI - Instruksi Upload"
echo "========================================"
echo ""
echo "File yang siap deploy ada di: $DIST/"
for goarch in "${BUILT[@]}"; do
    SIZE=$(du -h "$DIST/${APP_NAME}_linux_${goarch}" | cut -f1)
    echo "    ${APP_NAME}_linux_${goarch} (${SIZE})"
done
echo "    setup.sh            (wizard installer)"
echo "    templates/          (HTML templates)"
echo "    static/             (CSS, JS, icons)"
echo "    .env.example        (template konfigurasi)"
echo ""
echo "Pilih binary sesuai arsitektur VPS:"
echo "  amd64 = DigitalOcean / Vultr / Contabo / AWS (x86_64)"
echo "  arm64 = Raspberry Pi / Armbian / Oracle ARM"
echo ""
echo "========================================"
echo "  CARA DEPLOY"
echo "========================================"
echo ""
echo "1. Upload semua file ke VPS:"
for goarch in "${BUILT[@]}"; do
    echo "     scp $DIST/${APP_NAME}_linux_${goarch} root@IP_VPS:/opt/fintrack/"
done
echo "     scp $DIST/setup.sh root@IP_VPS:/opt/fintrack/"
echo "     scp -r $DIST/templates root@IP_VPS:/opt/fintrack/"
echo "     scp -r $DIST/static root@IP_VPS:/opt/fintrack/"
echo ""
echo "   Atau pakai rsync (lebih cepat):"
echo "     rsync -avz $DIST/ root@IP_VPS:/opt/fintrack/"
echo ""
echo "2. Di VPS, buat file .env dari template:"
echo "     cp .env.example .env"
echo "     nano .env   # isi JWT_SECRET, AI_API_KEY, dll"
echo ""
echo "3. Jalankan wizard installer:"
echo "     cd /opt/fintrack"
echo "     sudo bash setup.sh"
echo ""
echo "   Menu wizard:"
echo "     1) Install Baru   - setup systemd service, start app"
echo "     2) Update Binary  - backup + swap binary + restart"
echo "     3) Cek Status     - lihat status service + log"
echo "     4) Stop Service"
echo "     5) Keluar"
echo ""
echo "========================================"
echo "  TIPS"
echo "========================================"
echo "  - finance.db, .env, static/uploads/, static/avatars/ TIDAK diubah saat update"
echo "  - Binary lama di-backup saat update"
echo "  - dist/ di-gitignore (tidak masuk git)"
echo "  - Untuk SSL: setup nginx + certbot (lihat docs/buildtechnicl.md)"
echo ""
