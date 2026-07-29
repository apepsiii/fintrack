# FinTrack - Modern Minimalist Financial Tracker

![Version](https://img.shields.io/badge/version-1.0.0-brightgreen)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

**FinTrack** adalah aplikasi pencatatan keuangan pribadi berbasis web (PWA) dengan pendekatan antarmuka Modern Minimalist. Aplikasi ini dirancang untuk memecahkan masalah keengganan pengguna dalam mencatat pengeluaran karena UI aplikasi yang sering kali rumit.

## 🎯 Tujuan Produk

- **Frictionless Entry**: Memungkinkan pengguna mencatat pengeluaran dalam waktu kurang dari 5 detik
- **Clear Visibility**: Memberikan gambaran kondisi keuangan dalam satu kali lihat
- **Budget Awareness**: Mencegah pengeluaran berlebih melalui fitur target/anggaran dengan indikator visual yang intuitif

## ✨ Fitur Utama (MVP - Phase 1)

### ✅ Sudah Diimplementasikan

| Modul | Fitur | Deskripsi |
|-------|-------|-----------|
| **Authentication** | Splash Login | Halaman sapaan awal dan login berbasis Email/Password |
| **Dashboard** | Balance Overview | Menampilkan Saldo Tersedia dan Total Pengeluaran Bulan Ini |
| **Dashboard** | Recent Ledger | Menampilkan 5 transaksi terakhir |
| **Transactions** | Bottom Sheet Form | Modal interaktif dari navigasi bawah untuk mencatat In/Out |
| **Transactions** | OCR Receipt Scan | Memindai struk dengan kamera (Mockup/Simulasi) |
| **Analytics** | Visual Stats | Menampilkan persentase pengeluaran per kategori |
| **Budget** | Budget Monitoring | Pemantauan anggaran dengan progress bar berwarna |
| **Profile** | Settings Page | Halaman pengaturan profil dan aplikasi |

## 🛠️ Tech Stack

- **Backend**: Go 1.25 + Gin Web Framework
- **Database**: SQLite (CGO-free driver: modernc.org/sqlite)
- **Frontend**: HTMX + TailwindCSS + Phosphor Icons
- **Design System**: Modern Minimalist (Brand Color: #c3f545 Lime Green)

## 📁 Struktur Proyek

```
personalBudgetApps/
├── main.go                 # Backend server dengan Gin framework
├── go.mod                  # Go module dependencies
├── go.sum                  # Dependency checksums
├── finance.db             # SQLite database (auto-generated)
├── templates/
│   ├── index.html         # Dashboard - Saldo & Transaksi
│   ├── login.html         # Halaman Login & Splash
│   ├── stats.html         # Statistik & Analisis
│   ├── targets.html       # Pemantauan Anggaran
│   └── profile.html       # Profil & Pengaturan
└── docs/
    ├── PRD.md             # Product Requirements Document
    └── *.html             # Design mockups & references
```

## 🚀 Cara Menjalankan

### Prasyarat

- Go 1.21 atau lebih baru
- Git Bash atau terminal lainnya

### Instalasi & Menjalankan

```bash
# Clone atau masuk ke direktori proyek
cd E:/apep/web/personalBudgetApps

# Download dependencies
go mod tidy

# Jalankan server
go run main.go
```

Server akan berjalan di **http://localhost:8080**

### Build Executable

```bash
# Build binary
go build -o fintrack.exe

# Jalankan binary
./fintrack.exe
```

## 📱 Halaman Aplikasi

| Halaman | URL | Deskripsi |
|---------|-----|-----------|
| **Login** | `/login` | Autentikasi pengguna dengan email/password atau Google OAuth |
| **Dashboard** | `/` | Saldo tersedia, pengeluaran bulanan, 5 transaksi terakhir |
| **Statistik** | `/stats` | Breakdown pengeluaran per kategori dengan persentase |
| **Target Anggaran** | `/targets` | Monitoring budget dengan progress bar (hijau/kuning/merah) |
| **Profil** | `/profile` | Pengaturan akun dan aplikasi |

## 🗄️ Database Schema

### Tables

#### 1. `categories`
```sql
id          INTEGER PRIMARY KEY AUTOINCREMENT
name        TEXT NOT NULL           -- Nama kategori (Makan, Transport, dll)
type        TEXT NOT NULL           -- 'income' atau 'expense'
icon        TEXT NOT NULL           -- Icon class (ph-hamburger, ph-car)
```

**Data Awal (Seed)**:
- Makan (expense, ph-hamburger)
- Transport (expense, ph-car)
- Tagihan (expense, ph-lightning)
- Gaji (income, ph-money)

#### 2. `transactions`
```sql
id          INTEGER PRIMARY KEY AUTOINCREMENT
type        TEXT NOT NULL           -- 'income' atau 'expense'
amount      INTEGER NOT NULL        -- Nominal (Integer, tanpa desimal)
category_id INTEGER                 -- FK ke categories
date        DATETIME DEFAULT NOW    -- Tanggal transaksi
```

#### 3. `budgets`
```sql
id           INTEGER PRIMARY KEY AUTOINCREMENT
category_id  INTEGER NOT NULL UNIQUE -- FK ke categories
amount_limit INTEGER NOT NULL        -- Batas maksimal bulanan
```

**Data Awal (Seed)**:
- Makan: Rp 1,500,000
- Transport: Rp 500,000

## 🎨 Brand Colors

```css
--brand-lime: #c3f545      /* Primary accent */
--brand-lime-dark: #a3d124 /* Darker lime */
--brand-dark: #01381b      /* Dark green */
--brand-light: #f8faf7     /* Light background */
--brand-muted: #809b8d     /* Muted text */
```

## 🔌 API Endpoints

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `GET` | `/login` | Halaman login |
| `POST` | `/login` | Submit login (redirect ke `/`) |
| `GET` | `/` | Dashboard utama |
| `POST` | `/transaction` | Tambah transaksi baru |
| `GET` | `/stats` | Halaman statistik |
| `GET` | `/targets` | Halaman anggaran |
| `GET` | `/profile` | Halaman profil |
| `POST` | `/api/ocr` | OCR scan struk (simulasi, return HTML) |

## 🗺️ Roadmap

### Phase 2 (Q3 2026) - Kesiapan Produksi
- [ ] Menerapkan sistem Autentikasi sungguhan (JWT / Session Cookie)
- [ ] Mengubah aplikasi menjadi PWA resmi (manifest.json + Service Worker)
- [ ] Mengintegrasikan API Google Cloud Vision untuk OCR struk sungguhan

### Phase 3 (Q4 2026) - Fitur Lanjutan
- [ ] Fitur ekspor/unduh laporan keuangan ke CSV dan PDF
- [ ] Mode Offline First dengan sync otomatis
- [ ] Menambahkan Dark Mode sungguhan yang bisa di-toggle

## 🧪 Testing

Untuk mencoba aplikasi:

1. Buka browser dan akses http://localhost:8080/login
2. Klik "Masuk ke Akun" (autentikasi dummy, langsung masuk)
3. Di Dashboard, klik tombol **+** (floating action button) untuk menambah transaksi
4. Pilih nominal, kategori, dan jenis (Pengeluaran/Pemasukan)
5. Lihat hasilnya di Dashboard, Statistik, dan Target Anggaran

## 📝 Catatan Pengembangan

- **CGO-Free**: Menggunakan `modernc.org/sqlite` sebagai pengganti `mattn/go-sqlite3` untuk menghindari dependency CGO
- **HTMX**: Menggunakan HTMX untuk interaksi SPA-like tanpa JavaScript framework besar
- **Mobile-First**: Design responsif dengan fokus pada pengalaman mobile
- **Bottom Sheet Modal**: Inspired by native mobile apps untuk user experience yang familiar

## 📄 Lisensi

MIT License - Silakan gunakan untuk pembelajaran dan pengembangan pribadi.

## 👨‍💻 Developer

Dikembangkan sebagai proyek pembelajaran untuk memahami:
- Backend development dengan Go & Gin
- HTMX untuk interaksi dinamis
- SQLite sebagai embedded database
- Modern UI/UX design principles

---

**Versi**: 1.0.0 (MVP Phase 1)  
**Tanggal**: Juli 2026
