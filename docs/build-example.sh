#!/bin/bash
set -e

# ============================================================
#  SMK NIBA Super Apps - Build script
#
#  Builds linux binaries (CGO_ENABLED=0, stripped) into ./dist/
#  and prints upload instructions.
#
#  After upload, run setup.sh on the VPS for the wizard menu
#  (install / update / status).
#
#  Usage:
#    sh build.sh              # build amd64 + arm64
#    sh build.sh arm64        # build arm64 only
#    sh build.sh amd64        # build amd64 only
# ============================================================

VERSION="${VERSION:-v1.3.0}"
LDFLAGS="-s -w"
DIST="dist"

ARCH_ARG="${1:-both}"
case "$ARCH_ARG" in
    amd64) TARGETS=("amd64") ;;
    arm64) TARGETS=("arm64") ;;
    both)  TARGETS=("amd64" "arm64") ;;
    *) echo "Usage: sh build.sh [amd64|arm64|both]"; exit 1 ;;
esac

echo ""
echo "========================================"
echo "  SMK NIBA Super Apps - Build"
echo "  ${VERSION}  |  $(date +%Y-%m-%d)"
echo "========================================"

rm -rf "$DIST"
mkdir -p "$DIST"

BUILT=()
for goarch in "${TARGETS[@]}"; do
    OUT="$DIST/smartbell_linux_${goarch}"
    echo ""
    echo "[*] Building linux/${goarch} (CGO_ENABLED=0, stripped)..."
    GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 \
        go build -ldflags="${LDFLAGS}" -o "${OUT}" .
    SIZE=$(du -h "${OUT}" | cut -f1)
    echo "    -> ${SIZE}  ${OUT}"
    BUILT+=("$goarch")
done

echo ""
echo "========================================"
echo "  BUILD SELESAI - Instruksi Upload"
echo "========================================"
echo ""
echo "Binary + setup wizard ada di:"
echo "  $DIST/"
for goarch in "${BUILT[@]}"; do
    SIZE=$(du -h "$DIST/smartbell_linux_${goarch}" | cut -f1)
    echo "    smartbell_linux_${goarch} (${SIZE})"
done
cp setup.sh "$DIST/setup.sh" 2>/dev/null && echo "    setup.sh (wizard installer)"
echo ""
echo "Pilih binary sesuai architecture VPS:"
echo "  amd64 = DigitalOcean/Vultr/Contabo default (x86_64)"
echo "  arm64 = Raspberry Pi / Armbian / ARM cloud"
echo ""
echo "========================================"
echo "  CARA DEPLOY"
echo "========================================"
echo ""
echo "1. Upload 2 file ke VPS (folder baru atau folder existing):"
echo "   - smartbell_linux_<arch>  (binary)"
echo "   - setup.sh                (wizard)"
echo ""
echo "   Contoh SCP:"
for goarch in "${BUILT[@]}"; do
    echo "     scp $DIST/smartbell_linux_${goarch} root@IP_VPS:/opt/nibasuperapps/"
done
echo "     scp $DIST/setup.sh root@IP_VPS:/opt/nibasuperapps/"
echo ""
echo "2. Di VPS, jalankan wizard:"
echo "   cd /opt/nibasuperapps"
echo "   sudo bash setup.sh"
echo ""
echo "   Menu wizard:"
echo "     1) Install Baru   - setup systemd service, start app"
echo "     2) Update         - backup + swap binary + restart + health-check"
echo "     3) Cek Status     - lihat status service + log"
echo "     4) Keluar"
echo ""
echo "========================================"
echo "  TIPS"
echo "========================================"
echo "  - database.db, config.yaml, uploads/, logs/ TIDAK diubah saat update"
echo "  - Binary lama di-backup jadi smartbell.bak.<timestamp> saat update"
echo "  - dist/ di-gitignore (tidak masuk git)"
echo ""
