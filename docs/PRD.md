Product Requirements Document (PRD) & Entity Relationship Diagram (ERD)Project Name: FinTrack - Modern Minimalist Financial TrackerVersion: 1.0.0Date: Juli 20261. Product Requirements Document (PRD)1.1. Ringkasan Eksekutif (Executive Summary)FinTrack adalah aplikasi pencatatan keuangan pribadi berbasis web (PWA) dengan pendekatan antarmuka Modern Minimalist. Aplikasi ini dirancang untuk memecahkan masalah keengganan pengguna dalam mencatat pengeluaran karena UI aplikasi yang sering kali rumit. Dengan menggunakan stack teknologi ringan (Go, SQLite, HTMX), FinTrack menawarkan pengalaman pengguna layaknya aplikasi Native (SPA) dengan kecepatan akses instan.1.2. Tujuan Produk (Product Goals)Frictionless Entry: Memungkinkan pengguna mencatat pengeluaran dalam waktu kurang dari 5 detik (didukung fitur pemindai struk/OCR).Clear Visibility: Memberikan gambaran kondisi keuangan (saldo & pengeluaran bulanan) dalam satu kali lihat.Budget Awareness: Mencegah pengeluaran berlebih melalui fitur target/anggaran dengan indikator visual yang intuitif.1.3. Target Pengguna (Target Audience)Individu/Pekerja Profesional: Membutuhkan pelacakan gaji dan pengeluaran harian.Mobile-First Users: Pengguna yang lebih sering menggunakan ponsel untuk mencatat pengeluaran langsung di kasir/tempat belanja.1.4. Fitur Inti (Core Features)Berikut adalah daftar fitur utama yang disepakati untuk Fase MVP (Minimum Viable Product) menuju V1:ModulFiturDeskripsiStatusAuthenticationSplash LoginHalaman sapaan awal dan login berbasis Email/Password.✅ SelesaiOAuth GoogleIntegrasi login cepat menggunakan akun Google.⏳ TerencanaDashboardBalance OverviewMenampilkan Saldo Tersedia dan Total Pengeluaran Bulan Ini.✅ SelesaiRecent LedgerMenampilkan 5 transaksi terakhir.✅ SelesaiTransactionsBottom Sheet FormModal interaktif dari navigasi bawah untuk mencatat In/Out.✅ SelesaiOCR Receipt ScanMemindai struk dengan kamera dan otomatis mengekstrak nominal.🔶 Mockup (Simulasi)AnalyticsVisual StatsMenampilkan persentase pengeluaran per kategori.✅ SelesaiBudgetingMonthly TargetsMenetapkan batas pengeluaran kategori dengan progress bar (Aman/Waspada/Bahaya).✅ SelesaiSettingsUser ProfileMenampilkan info akun dan pengaturan ekspor data (CSV).✅ Selesai1.5. Spesifikasi Teknis (Tech Stack)Backend: Golang dengan framework Gin (Cepat, efisien memori).Frontend: HTML5 murni, Tailwind CSS (Styling), Phosphor Icons (Ikonografi).Interactivity: HTMX (Mengubah aplikasi menjadi SPA tanpa framework JS berat).Database: SQLite (Relasional, tanpa server khusus, mudah di-backup).2. Entity Relationship Diagram (ERD)Meskipun saat ini aplikasi kita berjalan secara lokal (1 database file), rancangan ERD di bawah ini sudah disiapkan dengan struktur Multi-User (Siap Skala). Artinya, jika suatu saat Anda mendeploy aplikasi ini ke server publik, data antar pengguna tidak akan tercampur.Catatan: Anda dapat menginstal ekstensi seperti "Markdown Preview Mermaid Support" di VS Code untuk melihat diagram di bawah ini sebagai visual grafis.erDiagram
    USERS ||--o{ TRANSACTIONS : "melakukan"
    USERS ||--o{ BUDGETS : "mengatur"
    CATEGORIES ||--o{ TRANSACTIONS : "mengelompokkan"
    CATEGORIES ||--o{ BUDGETS : "ditetapkan pada"

    USERS {
        int id PK "Auto Increment"
        string name "Nama lengkap pengguna"
        string email "Email unik pengguna"
        string password_hash "Kata sandi terenkripsi"
        datetime created_at "Tanggal registrasi"
    }

    CATEGORIES {
        int id PK "Auto Increment"
        string name "Nama Kategori (Makan, Gaji, dll)"
        string type "Enum: 'income' atau 'expense'"
        string icon "Class ikon (misal: 'ph-hamburger')"
    }

    TRANSACTIONS {
        int id PK "Auto Increment"
        int user_id FK "Relasi ke pengguna (Opsional di lokal)"
        int category_id FK "Relasi ke kategori"
        string type "Enum: 'income' atau 'expense'"
        int amount "Nominal uang (Integer, tanpa desimal)"
        string note "Catatan tambahan (Opsional)"
        datetime date "Tanggal transaksi (Default: Current)"
    }

    BUDGETS {
        int id PK "Auto Increment"
        int user_id FK "Relasi ke pengguna"
        int category_id FK "Relasi ke kategori"
        int amount_limit "Batas maksimal pengeluaran bulanan"
        string month_year "Format: YYYY-MM (misal: 2026-07)"
    }
2.1. Penjelasan Entitas & RelasiUSERS (Pengguna): Tabel utama untuk autentikasi. Satu user dapat memiliki banyak (One-to-Many) Transaksi dan Anggaran.CATEGORIES (Kategori): Tabel referensi/master. Satu kategori dapat digunakan di banyak transaksi maupun anggaran. Kita memisahkan entitas ini agar ikon dan nama kategori terpusat.TRANSACTIONS (Transaksi): Tabel operasional yang paling sering ditulis (Write-Heavy). Menyimpan aliran uang masuk dan keluar secara kronologis.BUDGETS (Anggaran/Target): Menyimpan batas maksimal pengeluaran per user, per kategori, pada bulan dan tahun tertentu.3. Peta Jalan Pengembangan (Roadmap)Fase 2 (Q3 2026) - Kesiapan Produksi[ ] Menerapkan sistem Autentikasi sungguhan (JWT / Session Cookie di Gin).[ ] Mengubah aplikasi menjadi PWA resmi (menambahkan manifest.json dan Service Worker agar bisa di-install di HP).[ ] Mengintegrasikan API Google Cloud Vision untuk menggantikan fitur simulasi (Mockup) pemindai OCR struk.Fase 3 (Q4 2026) - Fitur Lanjutan[ ] Fitur ekspor/unduh laporan keuangan ke format CSV dan PDF.[ ] Mode Offline First: Memungkinkan pengguna mencatat saat tidak ada internet, dan melakukan sync saat terhubung kembali.[ ] Menambahkan Dark Mode sungguhan yang bisa di-toggle dari halaman Profil.