# FinTrack — Modern Minimalist Financial Tracker

> Go + Gin + SQLite + HTMX + TailwindCSS  
> Version: 3.x | Status: Production Ready

---

## Quick Start

```bash
# Jalankan server
go run main.go
# atau gunakan binary
./fintrack.exe

# Akses aplikasi
http://localhost:8080
```

Login default: `admin@fintrack.id` / `admin123`

---

## Tech Stack

| Layer | Teknologi |
|-------|-----------|
| Backend | Go 1.25 + Gin 1.9.1 |
| Database | SQLite (modernc.org/sqlite, CGO-free) |
| Frontend | HTMX 1.9.10 + TailwindCSS (CDN) |
| Auth | JWT (HS256) + bcrypt |
| Icons | Phosphor Icons |
| Font | Inter (Google Fonts) |
| PWA | manifest.json + service-worker.js |

---

## Struktur File

```
fintrack/
├── main.go               # Server utama, routes, handlers
├── auth.go               # JWT, bcrypt, register/login
├── middleware.go         # Auth middleware
├── migrations.go         # Migrasi database otomatis
├── ocr.go                # OCR.Space + AI parser struk
├── export.go             # CSV/PDF export
├── savings.go            # Tabungan & goals
├── categories.go         # Kategori custom
├── debt.go               # Hutang & piutang
├── recurring.go          # Transaksi berulang
├── ai.go                 # FinBot AI (OpenAI-compatible)
├── forgot_password.go    # Reset password flow
├── embed.go              # Embed static assets
├── go.mod / go.sum
├── .env.example          # Template konfigurasi
├── finance.db            # SQLite database (runtime)
├── fintrack.exe          # Binary Windows
│
├── templates/            # Go HTML templates
│   ├── index.html        # Dashboard
│   ├── login.html        # Authentication
│   ├── stats.html        # Statistik & tren
│   ├── targets.html      # Monitor anggaran
│   ├── profile.html      # Profil & pengaturan
│   ├── history.html      # Riwayat transaksi
│   ├── savings.html      # Daftar tabungan
│   ├── savings_detail.html
│   ├── categories.html   # Kelola kategori
│   ├── recurring.html    # Transaksi berulang
│   ├── debt.html         # Hutang & piutang
│   ├── debt_detail.html
│   ├── budget_manage.html
│   ├── ai.html           # FinBot AI chat
│   └── export_pdf.html
│
└── static/
    ├── manifest.json     # PWA manifest
    ├── service-worker.js # Offline support
    ├── offline.js        # IndexedDB & sync
    └── dark-mode.css     # Dark theme
```

---

## Halaman & URL

| Halaman | URL | Deskripsi |
|---------|-----|-----------|
| Dashboard | `/` | Saldo, transaksi terakhir |
| Login | `/login` | Autentikasi |
| Statistik | `/stats` | Breakdown pengeluaran |
| Anggaran | `/targets` | Monitor budget |
| Profil | `/profile` | Pengaturan & ekspor |
| Riwayat | `/history` | Semua transaksi + filter |
| Tabungan | `/savings` | Goals tabungan |
| Hutang | `/debt` | Hutang & piutang |
| Kategori | `/categories` | Kelola kategori |
| Berulang | `/recurring` | Transaksi otomatis |
| AI | `/ai` | FinBot AI chat |
| Budget | `/budget/manage` | CRUD anggaran |

---

## Fitur Utama

### Transaksi
- Tambah pemasukan/pengeluaran via bottom sheet modal
- Scan struk: OCR.Space → AI parser → popup konfirmasi → isi form otomatis
- Edit & delete transaksi
- Upload foto struk
- Catatan otomatis dari hasil scan

### Anggaran
- Set limit per kategori per bulan
- Progress bar berwarna (hijau/kuning/merah)
- Full CRUD via `/budget/manage`

### Tabungan
- 15 tipe goals (darurat, haji, rumah, investasi, dll)
- Deposit & withdraw dengan linked transaction
- Tracking progress target

### Hutang & Piutang
- Catat hutang (saya berhutang) & piutang (saya memberi hutang)
- Riwayat pembayaran

### Transaksi Berulang
- Jadwal otomatis: harian, mingguan, bulanan, tahunan
- Auto-generate transaksi saat jatuh tempo

### Kategori
- 22 kategori default (15 pengeluaran + 7 pemasukan)
- Tambah kategori custom
- Tombol "Generate Default" untuk restore kategori bawaan

### AI (FinBot)
- Chat asisten keuangan berbasis OpenAI-compatible API
- Analisis data keuangan user secara real-time

### Ekspor & Impor
- CSV: transaksi, anggaran, ringkasan
- PDF: laporan lengkap
- JSON: backup & restore semua data

### Reset Data
- Pilih data mana yang ingin dihapus (transaksi, anggaran, tabungan, kategori, hutang, berulang)
- Konfirmasi ketik "HAPUS" sebelum eksekusi

### PWA
- Installable di Android & desktop
- Offline support via service worker + IndexedDB
- Background sync saat kembali online

---

## Konfigurasi (.env)

```env
SERVER_PORT=8080
GIN_MODE=release
JWT_SECRET=your-secret-key
JWT_EXPIRY_HOURS=24
DB_PATH=./finance.db
BCRYPT_COST=10

# OCR (scan struk)
OCR_SPACE_API_KEY=your-key   # https://ocr.space/ocrapi

# AI (FinBot + OCR parser)
AI_PROVIDER=openai
AI_API_KEY=your-key
AI_MODEL=gpt-4o-mini
AI_BASE_URL=https://api.openai.com/v1
AI_MAX_TOKENS=1000
AI_TEMPERATURE=0.7
```

---

## API Endpoints

### Transaksi
```
POST   /transaction
PUT    /api/transaction/:id
DELETE /api/transaction/:id
GET    /api/transaction/:id
```

### Kategori
```
GET    /categories
POST   /api/categories
PUT    /api/categories/:id
DELETE /api/categories/:id
POST   /api/categories/generate-default
```

### Anggaran
```
POST   /api/budget
DELETE /api/budget
GET    /api/budgets
GET    /api/budget/:id
```

### Tabungan
```
GET    /savings
GET    /savings/:id
POST   /api/savings
PUT    /api/savings/:id
DELETE /api/savings/:id
POST   /api/savings/:id/transaction
```

### Hutang
```
GET    /debt
GET    /debt/:id
POST   /api/debt
PUT    /api/debt/:id
DELETE /api/debt/:id
POST   /api/debt/:id/pay
```

### Berulang
```
GET    /recurring
POST   /api/recurring
PUT    /api/recurring/:id
DELETE /api/recurring/:id
```

### Ekspor
```
GET  /export/transactions/csv
GET  /export/budget/csv
GET  /export/summary/csv
GET  /export/pdf
GET  /export/backup
POST /import/backup
```

### Lainnya
```
POST /api/ocr
POST /api/reset-data
POST /api/ai/analyze
GET  /api/ai/chat
PUT  /api/profile
POST /api/avatar/preset
POST /api/avatar/upload
```

---

## Database Schema

```sql
users           (id, name, email, password_hash, avatar, created_at)
transactions    (id, user_id, type, amount, category_id, note, date, receipt_path)
categories      (id, name, type, icon, user_id, is_default)
budgets         (id, user_id, category_id, amount_limit, month_year)
savings         (id, user_id, name, type, icon, color, target_amount, current_amount, deadline, is_completed)
savings_transactions (id, savings_id, user_id, amount, type, note, date, linked_transaction_id)
debts           (id, user_id, name, type, creditor, total_amount, paid_amount, due_date, is_settled)
debt_payments   (id, debt_id, amount, note, date)
recurring_transactions (id, user_id, name, type, amount, category_id, frequency, next_date, is_active)
schema_migrations (version, name, applied_at)
password_resets (id, user_id, token, expires_at, used)
```

---

## Build & Deploy

```bash
# Build binary
go build -o fintrack.exe .

# Tidy dependencies
go mod tidy

# Reset database (hapus & buat ulang)
del finance.db
.\fintrack.exe
```

---

## Design System

| Token | Value |
|-------|-------|
| Brand Lime | `#c3f545` |
| Brand Dark | `#01381b` |
| Background | `#f1f5f9` |
| Muted | `#809b8d` |
| Max Width | 448px (mobile-first) |
| Font | Inter 400–800 |

---

## Troubleshooting

**Port sudah dipakai** → Set `SERVER_PORT=8081` di `.env`

**Database error** → Hapus `finance.db`, restart server

**OCR tidak akurat** → Pastikan `OCR_SPACE_API_KEY` dan `AI_API_KEY` diset di `.env`. Foto struk harus terang dan tidak blur.

**Kategori kosong** → Buka `/categories` → klik tombol "Default"

**AI tidak merespons** → Cek `AI_API_KEY` dan `AI_BASE_URL` di `.env`
