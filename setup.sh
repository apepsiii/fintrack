#!/bin/bash

# ============================================================
#  FinTrack - Setup Wizard
#
#  All-in-one installer untuk install baru, update binary,
#  dan cek status di VPS Linux.
#
#  Cara pakai:
#    sudo bash setup.sh
# ============================================================

APP_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_NAME="fintrack"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SERVICE_NAME="$APP_NAME"

echo "========================================="
echo "  FinTrack - Setup Wizard"
echo "========================================="
echo "  App folder  : $APP_DIR"
echo "  Service name: $SERVICE_NAME"
echo ""

if [ "$EUID" -ne 0 ]; then
    echo "Harap jalankan dengan sudo: sudo bash setup.sh"
    exit 1
fi

# â”€â”€ Deteksi binary â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
detect_binary() {
    local arch
    arch=$(uname -m 2>/dev/null)
    case "$arch" in
        x86_64)        arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
    esac
    # Cari binary fintrack
    for name in fintrack "${APP_NAME}"; do
        [ -f "$APP_DIR/$name" ] && [ -x "$APP_DIR/$name" ] && echo "$name" && return
    done
    # Cari binary dengan nama arch
    local found
    found=$(ls -t "$APP_DIR/${APP_NAME}_linux_${arch}" 2>/dev/null | head -1)
    [ -n "$found" ] && basename "$found" && return
    echo ""
}

# â”€â”€ Install Baru â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
install_app() {
    BINARY=$(detect_binary)
    if [ -z "$BINARY" ]; then
        echo "[ERROR] Binary tidak ditemukan di $APP_DIR"
        echo "        Upload dulu binary fintrack_linux_amd64 atau fintrack_linux_arm64"
        exit 1
    fi

    read -p "Port aplikasi [8080]: " PORT
    PORT="${PORT:-8080}"

    # Buat folder yang diperlukan
    mkdir -p "$APP_DIR/static/uploads"
    mkdir -p "$APP_DIR/static/avatars"

    # Pastikan .env ada
    if [ ! -f "$APP_DIR/.env" ]; then
        if [ -f "$APP_DIR/.env.example" ]; then
            cp "$APP_DIR/.env.example" "$APP_DIR/.env"
            echo "[INFO] .env dibuat dari .env.example"
        else
            # Buat .env minimal
            cat > "$APP_DIR/.env" <<ENVEOF
SERVER_PORT=${PORT}
JWT_SECRET=$(openssl rand -hex 32)
DB_PATH=./finance.db
GIN_MODE=release
BCRYPT_COST=10
JWT_EXPIRY_HOURS=24
ENVEOF
            echo "[INFO] .env minimal dibuat otomatis"
        fi
    fi

    # Set port di .env
    sed -i "s/^SERVER_PORT=.*/SERVER_PORT=${PORT}/" "$APP_DIR/.env"

    # Pastikan JWT_SECRET tidak kosong
    if grep -q "JWT_SECRET=$" "$APP_DIR/.env" || grep -q "JWT_SECRET=your" "$APP_DIR/.env"; then
        NEW_SECRET=$(openssl rand -hex 32)
        sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${NEW_SECRET}/" "$APP_DIR/.env"
        echo "[INFO] JWT_SECRET di-generate otomatis"
    fi

    chmod +x "$APP_DIR/$BINARY"

    # Buat systemd service
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=FinTrack Personal Finance App
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${APP_DIR}
ExecStart=${APP_DIR}/${BINARY}
Restart=always
RestartSec=5
EnvironmentFile=${APP_DIR}/.env
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo ""
        echo "[OK] Instalasi berhasil!"
        echo "     App berjalan di: http://localhost:${PORT}"
        echo "     Default login  : admin@fintrack.id / admin123"
        echo "     PENTING        : Ganti password default setelah login!"
        echo ""
        echo "Log: journalctl -u $SERVICE_NAME -f"
    else
        echo "[ERROR] App gagal start. Cek log:"
        journalctl -u "$SERVICE_NAME" -n 30 --no-pager
    fi
}

# â”€â”€ Update Binary â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
update_app() {
    BINARY=$(detect_binary)
    if [ -z "$BINARY" ]; then
        echo "[ERROR] Binary tidak ditemukan."
        exit 1
    fi

    TS=$(date +%Y%m%d_%H%M%S)

    # Backup binary lama
    RUNNING_BIN=$(systemctl show -p ExecStart "$SERVICE_NAME" 2>/dev/null | sed 's/ExecStart=//;s/ .*//')
    if [ -f "$RUNNING_BIN" ]; then
        cp "$RUNNING_BIN" "${RUNNING_BIN}.bak.${TS}"
        echo "[INFO] Backup binary lama: ${RUNNING_BIN}.bak.${TS}"
    fi

    chmod +x "$APP_DIR/$BINARY"

    # Update path binary di service jika berubah
    sed -i "s|ExecStart=.*|ExecStart=${APP_DIR}/${BINARY}|" "$SERVICE_FILE"

    systemctl daemon-reload
    systemctl restart "$SERVICE_NAME"

    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "[OK] Update berhasil! Service di-restart."
        echo "     Binary aktif: $BINARY"
    else
        echo "[ERROR] Gagal restart. Cek log:"
        journalctl -u "$SERVICE_NAME" -n 30 --no-pager
        # Rollback
        if [ -f "${RUNNING_BIN}.bak.${TS}" ]; then
            echo "[INFO] Mencoba rollback ke binary lama..."
            cp "${RUNNING_BIN}.bak.${TS}" "$RUNNING_BIN"
            chmod +x "$RUNNING_BIN"
            systemctl restart "$SERVICE_NAME"
            sleep 2
            if systemctl is-active --quiet "$SERVICE_NAME"; then
                echo "[OK] Rollback berhasil."
            else
                echo "[ERROR] Rollback gagal. Cek manual."
            fi
        fi
    fi
}

# â”€â”€ Cek Status â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
check_status() {
    echo ""
    echo "=== Status Service ==="
    systemctl status "$SERVICE_NAME" --no-pager
    echo ""
    echo "=== Log Terbaru (30 baris) ==="
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager
    echo ""
    echo "=== Disk Usage ==="
    du -sh "$APP_DIR" 2>/dev/null
    if [ -f "$APP_DIR/finance.db" ]; then
        echo "Database: $(du -sh "$APP_DIR/finance.db" | cut -f1)"
    fi
}

# â”€â”€ Setup Nginx + SSL â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
setup_nginx() {
    read -p "Domain/subdomain (misal: fintrack.domain.com): " DOMAIN
    if [ -z "$DOMAIN" ]; then
        echo "Domain tidak boleh kosong."
        return
    fi

    read -p "Port app [8080]: " APP_PORT
    APP_PORT="${APP_PORT:-8080}"

    # Install nginx & certbot jika belum ada
    if ! command -v nginx &>/dev/null; then
        echo "[INFO] Menginstall nginx..."
        apt-get update -q && apt-get install -y nginx
    fi

    NGINX_CONF="/etc/nginx/sites-available/${APP_NAME}"
    cat > "$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    client_max_body_size 20M;

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF

    ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/${APP_NAME}"
    nginx -t && systemctl reload nginx

    echo "[OK] Nginx dikonfigurasi untuk $DOMAIN -> port $APP_PORT"
    echo ""

    read -p "Setup SSL dengan Certbot? (y/n): " DO_SSL
    if [ "$DO_SSL" = "y" ]; then
        if ! command -v certbot &>/dev/null; then
            apt-get install -y certbot python3-certbot-nginx
        fi
        certbot --nginx -d "$DOMAIN"
    fi
}

# â”€â”€ Main Menu â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
echo "Pilih aksi:"
echo "  1) Install Baru"
echo "  2) Update Binary"
echo "  3) Cek Status & Log"
echo "  4) Setup Nginx + SSL"
echo "  5) Stop Service"
echo "  6) Keluar"
echo ""
read -p "Pilihan [1-6]: " CHOICE

case "$CHOICE" in
    1) install_app ;;
    2) update_app ;;
    3) check_status ;;
    4) setup_nginx ;;
    5) systemctl stop "$SERVICE_NAME" && echo "[OK] Service dihentikan." ;;
    6) exit 0 ;;
    *) echo "Pilihan tidak valid." ;;
esac
