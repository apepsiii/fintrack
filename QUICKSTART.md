# Quick Start Guide

## 🚀 Memulai Aplikasi

### Opsi 1: Menggunakan Script (Recommended)

**Windows:**
```bash
start.bat
```

**Linux/Mac:**
```bash
chmod +x start.sh
./start.sh
```

### Opsi 2: Manual

```bash
# Download dependencies (only first time)
go mod tidy

# Run server
go run main.go
```

## 📊 Menambahkan Data Sample

Untuk testing dengan data sample yang sudah disiapkan:

```bash
# Run seeder
go run seed.go

# Then start the server
go run main.go
```

Data sample akan mengisi:
- 2 transaksi pemasukan (Gaji)
- 19 transaksi pengeluaran di berbagai kategori
- Data tersebar dalam 20 hari terakhir

## 🌐 Mengakses Aplikasi

Setelah server berjalan, buka browser:

1. **Halaman Login**  
   http://localhost:8080/login
   - Klik "Masuk ke Akun" (bypass authentication)

2. **Dashboard**  
   http://localhost:8080/
   - Lihat saldo dan transaksi terakhir
   - Klik tombol **+** untuk tambah transaksi

3. **Statistik**  
   http://localhost:8080/stats
   - Breakdown pengeluaran per kategori

4. **Target Anggaran**  
   http://localhost:8080/targets
   - Monitor budget dengan progress bar

5. **Profil**  
   http://localhost:8080/profile
   - Pengaturan akun

## ✨ Fitur yang Bisa Dicoba

### Menambah Transaksi
1. Di Dashboard, klik tombol **+** (floating button hijau)
2. Masukkan nominal (atau klik icon "Scan" untuk simulasi OCR)
3. Pilih kategori dari dropdown
4. Pilih jenis: **Pengeluaran** (merah) atau **Pemasukan** (hijau)
5. Lihat hasilnya langsung di Dashboard

### Simulasi OCR (Mockup)
1. Di modal transaksi, klik icon **Scan**
2. Upload gambar apa saja (simulasi)
3. Setelah 1.5 detik, nominal Rp 85.000 akan muncul otomatis
4. Nominal bisa diedit manual sebelum submit

### Melihat Statistik
1. Klik tab **Statistik** di bottom navigation
2. Lihat total pengeluaran bulan ini
3. Lihat breakdown per kategori dengan bar chart

### Monitor Budget
1. Klik tab **Target** di bottom navigation
2. Lihat progress bar untuk setiap kategori
3. Warna berubah sesuai usage:
   - **Hijau**: < 80% (aman)
   - **Kuning**: 80-99% (warning)
   - **Merah**: ≥ 100% (over budget)

## 🗄️ Database

Database SQLite akan dibuat otomatis di:
```
E:\apep\web\personalBudgetApps\finance.db
```

Untuk reset database, cukup hapus file `finance.db` dan restart server.

## 🛑 Menghentikan Server

Tekan **Ctrl+C** di terminal untuk stop server.

## 🐛 Troubleshooting

### Port 8080 Sudah Digunakan
Jika port 8080 sudah dipakai aplikasi lain, edit `main.go` baris terakhir:
```go
router.Run(":8080")  // Ganti 8080 ke port lain, misal :8081
```

### Database Error
Hapus `finance.db` dan restart server untuk membuat database baru.

### Templates Not Found
Pastikan folder `templates/` ada dan berisi 5 file HTML:
- index.html
- login.html
- stats.html
- targets.html
- profile.html

## 📱 Tips Penggunaan

1. **Mobile Testing**: Buka di Chrome DevTools dengan device emulation untuk melihat pengalaman mobile
2. **Data Persistence**: Semua transaksi disimpan permanen di SQLite
3. **Kategori**: Kategori default sudah disiapkan, untuk fase berikutnya bisa menambah custom categories

## 🔗 Next Steps

Setelah familiar dengan aplikasi:
1. Coba tambah berbagai jenis transaksi
2. Monitor bagaimana statistik berubah
3. Lihat budget warning saat mendekati limit
4. Eksplorasi code di `main.go` untuk memahami backend logic
5. Modifikasi templates HTML untuk custom UI

---

Selamat mencoba! 🎉
