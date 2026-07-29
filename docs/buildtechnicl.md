# Build & Deploy Guide — Go Web App

Panduan ini menjelaskan cara build, deploy, dan mengelola Go web app ke VPS Linux menggunakan pola yang sudah terbukti.

---

## Struktur yang Diperlukan

```
project/
├── main.go              # Entry point
├── go.mod
├── go.sum
├── config.yaml          # Konfigurasi (dibuat otomatis saat pertama jalan)
├── build.sh             # Build script (lihat di bawah)
├── setup.sh             # Setup wizard VPS (lihat di bawah)
├── views/               # HTML templates (embed atau folder)
├── public/              # Static files
└── dist/                # Output build (di-gitignore)
```

---

## 1. Build Script (`build.sh`)

Letakkan di root project. Build menggunakan `CGO_ENABLED=0` agar binary portable tanpa libc dependency.

```bash
#!/bin/bash
set -e

VERSION="${VERSION:-v1.0.0}"
LDFLAGS="-s -w"
DIST="dist"
APP_NAME="myapp"   # <-- ganti sesuai nama project

ARCH_ARG="${1:-both}"
case "$ARCH_ARG" in
    amd64) TARGETS=("amd64") ;;
    arm64) TARGETS=("arm64") ;;
    both)  TARGETS=("amd64" "arm64") ;;
    *) echo "Usage: sh build.sh [amd64|arm64|both]"; exit 1 ;;
esac

echo "========================================"
echo "  ${APP_NAME} - Build ${VERSION}"
echo "  $(date +%Y-%m-%d)"
echo "========================================"

rm -rf "$DIST"
mkdir -p "$DIST"

for goarch in "${TARGETS[@]}"; do
    OUT="$DIST/${APP_NAME}_linux_${goarch}"
    echo "[*] Building linux/${goarch}..."
    GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 \
        go build -ldflags="${LDFLAGS}" -o "${OUT}" .
    SIZE=$(du -h "${OUT}" | cut -f1)
    echo "    -> ${SIZE}  ${OUT}"
done

cp setup.sh "$DIST/setup.sh" 2>/dev/null && echo "    setup.sh copied"

echo ""
echo "Binary ada di dist/"
echo "Upload ke VPS lalu jalankan: sudo bash setup.sh"
```

**Cara pakai:**
```bash
sh build.sh          # build amd64 + arm64
sh build.sh amd64    # amd64 saja
sh build.sh arm64    # arm64 saja (Raspberry Pi, ARM VPS)
```

---

## 2. Setup Wizard (`setup.sh`)

All-in-one installer untuk install baru, update binary, dan cek status. Letakkan di root project, ikut di-copy ke `dist/` oleh `build.sh`.

```bash
#!/bin/bash

APP_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_NAME="$(basename "$APP_DIR")"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SERVICE_NAME="$APP_NAME"

echo "========================================="
echo "  ${APP_NAME} - Setup Wizard"
echo "========================================="
echo "  App folder  : $APP_DIR"
echo "  Service name: $SERVICE_NAME"
echo ""

if [ "$EUID" -ne 0 ]; then
  echo "Harap jalankan dengan sudo: sudo bash setup.sh"
  exit 1
fi

# Deteksi binary
detect_binary() {
    local arch
    arch=$(uname -m 2>/dev/null)
    case "$arch" in
        x86_64)        arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
    esac
    for c in app myapp smartbell; do
        [ -f "$APP_DIR/$c" ] && [ -x "$APP_DIR/$c" ] && echo "$c" && return
    done
    ls -t "$APP_DIR"/*_linux_${arch} 2>/dev/null | head -1 | xargs basename 2>/dev/null
}

install_app() {
    BINARY=$(detect_binary)
    if [ -z "$BINARY" ]; then
        echo "Binary tidak ditemukan. Upload dulu binary ke $APP_DIR"
        exit 1
    fi

    read -p "Port aplikasi [8080]: " PORT
    PORT="${PORT:-8080}"

    chmod +x "$APP_DIR/$BINARY"

    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=${APP_NAME}
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${APP_DIR}
ExecStart=${APP_DIR}/${BINARY}
Restart=always
RestartSec=5
Environment=PORT=${PORT}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "Instalasi berhasil! App berjalan di port $PORT"
    else
        echo "App gagal start. Cek log: journalctl -u $SERVICE_NAME -n 50"
    fi
}

update_app() {
    BINARY=$(detect_binary)
    if [ -z "$BINARY" ]; then
        echo "Binary tidak ditemukan."
        exit 1
    fi

    TS=$(date +%Y%m%d_%H%M%S)
    RUNNING=$(systemctl show -p ExecStart "$SERVICE_NAME" 2>/dev/null | grep -o '[^ ]*$')
    [ -f "$RUNNING" ] && cp "$RUNNING" "${RUNNING}.bak.${TS}" && echo "Backup: ${RUNNING}.bak.${TS}"

    chmod +x "$APP_DIR/$BINARY"

    # Update binary path di service jika berbeda
    sed -i "s|ExecStart=.*|ExecStart=${APP_DIR}/${BINARY}|" "$SERVICE_FILE"

    systemctl daemon-reload
    systemctl restart "$SERVICE_NAME"

    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "Update berhasil! Service di-restart."
    else
        echo "Gagal restart. Cek log: journalctl -u $SERVICE_NAME -n 50"
    fi
}

check_status() {
    echo ""
    systemctl status "$SERVICE_NAME" --no-pager
    echo ""
    echo "Log terbaru:"
    journalctl -u "$SERVICE_NAME" -n 20 --no-pager
}

echo "Pilih aksi:"
echo "  1) Install Baru"
echo "  2) Update Binary"
echo "  3) Cek Status & Log"
echo "  4) Stop Service"
echo "  5) Keluar"
echo ""
read -p "Pilihan [1-5]: " CHOICE

case "$CHOICE" in
    1) install_app ;;
    2) update_app ;;
    3) check_status ;;
    4) systemctl stop "$SERVICE_NAME" && echo "Service dihentikan." ;;
    5) exit 0 ;;
    *) echo "Pilihan tidak valid." ;;
esac
```

---

## 3. Deploy ke VPS — Step by Step

### Build di lokal (Windows)
```powershell
# PowerShell
$Env:GOOS = "linux"
$Env:GOARCH = "amd64"
$Env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o dist/myapp_linux_amd64 .

# Atau pakai build.sh via Git Bash
sh build.sh amd64
```

### Upload ke VPS
```bash
# SCP
scp dist/myapp_linux_amd64 root@IP_VPS:/opt/myapp/
scp dist/setup.sh root@IP_VPS:/opt/myapp/

# Atau rsync (lebih cepat untuk banyak file)
rsync -avz dist/ views/ public/ config.yaml root@IP_VPS:/opt/myapp/
```

### Install di VPS
```bash
ssh root@IP_VPS
cd /opt/myapp
sudo bash setup.sh
# Pilih: 1) Install Baru
# Masukkan port misal 8080
```

### Update binary (tanpa downtime lama)
```bash
# Upload binary baru
scp dist/myapp_linux_amd64 root@IP_VPS:/opt/myapp/

# SSH ke VPS
ssh root@IP_VPS
cd /opt/myapp
sudo bash setup.sh
# Pilih: 2) Update Binary
```

---

## 4. Reverse Proxy dengan Nginx + SSL

```bash
# Install nginx
sudo apt install nginx certbot python3-certbot-nginx -y

# Buat config
cat > /etc/nginx/sites-available/myapp <<EOF
server {
    listen 80;
    server_name domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_cache_bypass \$http_upgrade;
    }

    client_max_body_size 50M;
}
EOF

ln -s /etc/nginx/sites-available/myapp /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

# SSL dengan Certbot
certbot --nginx -d domain.com
```

---

## 5. Tips & Best Practices

### .gitignore
```gitignore
dist/
*.exe
database.db
*.db
config.yaml
uploads/
logs/
*.bak.*
```

### Embedded files (agar binary standalone)
```go
//go:embed views/* public/*
var staticFiles embed.FS
```
Binary akan membawa semua file views dan public, sehingga hanya perlu upload 1 file binary saja.

### Environment variables
```bash
# Di systemd service, tambahkan:
Environment=PORT=8080
Environment=DB_PATH=/opt/myapp/data.db
Environment=APP_ENV=production
```

### Health check setelah deploy
```bash
curl -f http://localhost:8080/health || echo "App not responding"
```

### Log management
```bash
# Lihat log realtime
journalctl -u myapp -f

# Log 100 baris terakhir
journalctl -u myapp -n 100

# Log hari ini
journalctl -u myapp --since today
```

---

## 6. Checklist Deploy

- [ ] Build binary: `sh build.sh amd64`
- [ ] Upload binary + setup.sh ke VPS
- [ ] Jalankan `sudo bash setup.sh` → Install Baru / Update
- [ ] Verifikasi service running: `systemctl status myapp`
- [ ] Test endpoint: `curl http://localhost:PORT`
- [ ] Setup nginx reverse proxy
- [ ] Setup SSL dengan certbot
- [ ] Cek log: `journalctl -u myapp -n 50`

---

## 7. Troubleshooting

| Masalah | Solusi |
|---------|--------|
| Binary tidak executable | `chmod +x binary_name` |
| Port already in use | `ss -tlnp \| grep PORT`, kill proses lama |
| Service tidak start | `journalctl -u myapp -n 50 --no-pager` |
| Permission denied | Pastikan `WorkingDirectory` dan file uploads writable |
| SQLite locked | Pastikan hanya 1 instance berjalan |
| CGO error di VPS | Pastikan build dengan `CGO_ENABLED=0` |
