# Struktur Proyek ZenGoBox

Dokumen ini menjelaskan dua struktur utama dari proyek ZenGoBox:
1. **Struktur Kode Sumber Backend (Golang)**: Menjelaskan bagaimana kode sumber aplikasi proksi ini diorganisasikan.
2. **Struktur Modul Magisk/KernelSU**: Menjelaskan bagaimana hasil kompilasi dibungkus dan ditanamkan ke dalam OS Android.

---

## 1. Struktur Kode Sumber Backend (Golang)

Ini adalah susunan folder dari kode asli (sebelum dikompilasi). Dirancang secara modular dan rapi.

```text
zengobox/
├── cmd/
│   └── zengobox/
│       ├── main.go         # Titik masuk utama program, penangkapan versi
│       └── commands.go     # Berisi semua perintah CLI (start, stop, daemon, update, toggle)
│
├── internal/
│   ├── config/
│   │   ├── config.go       # Mendefinisikan struktur YAML dan logika baca/tulis config
│   │   └── default.go      # Berisi setelan bawaan jika zengobox.yaml belum ada
│   │
│   ├── core/
│   │   ├── manager.go      # Skrip untuk Menjalankan, Membunuh (Kill), dan Memantau proses Mihomo/Sing-box
│   │   └── scheduler.go    # Mesin Cron bawaan untuk auto-update langganan & geo di latar belakang
│   │
│   ├── logger/
│   │   └── log.go          # Menulis file runs.log dan menampilkan Notifikasi Toast di Android
│   │
│   ├── netfilter/          # MESIN UTAMA IPTABLES
│   │   ├── ipt.go          # Wrapper iptables yang mencegah tabrakan (resource busy)
│   │   ├── mode.go         # Penyeleksi mode (tproxy, redirect, dll) & fitur CleanAllNetfilter
│   │   ├── rules.go        # Pemrosesan daftar IP bypass, UID aplikasi, dll
│   │   ├── tproxy.go       # Logika injeksi rantai mangle untuk TPROXY murni
│   │   ├── redirect.go     # Logika injeksi rantai nat untuk TCP REDIRECT murni
│   │   ├── enhance.go      # Logika hibrida (Redirect TCP + Tproxy UDP)
│   │   └── dns.go          # Sistem penculikan DNS (DNS Hijacking) ke port proxy
│   │
│   ├── network/
│   │   ├── watcher.go      # Pembaca perubahan antarmuka WiFi & Hotspot (Tethering)
│   │   └── module.go       # Mendeteksi jika Anda mematikan modul lewat aplikasi Magisk
│   │
│   ├── updater/
│   │   ├── download.go     # Mesin pengunduh file cerdas dengan fungsi Resume & auto-Mirror
│   │   ├── github.go       # Berkomunikasi dengan API Github untuk mencari versi terbaru
│   │   ├── kernel.go       # Logika khusus untuk memperbarui binary Mihomo / Sing-box
│   │   ├── geo.go          # Logika khusus untuk memperbarui database geoip & geosite
│   │   └── dashboard.go    # Pengekstrak file zip UI (Zashboard, Yacd, dll) dari Github
│   │
│   └── web/
│       ├── server.go       # Backend Server (API) untuk Web UI Zashboard di port 9999
│       └── assets/         # File statis (app.js, index.html, style.css) yang ditanam ke dalam program Go
│
├── Makefile                # Kumpulan skrip kompilasi otomatis (build-magisk)
└── README.md               # Panduan penggunaan
```

---

## 2. Struktur Instalasi Modul Magisk (Hasil Kompilasi)

Saat Anda mengunduh file `.zip` ZenGoBox dan mem-*flash*-nya di Magisk/KernelSU, sistem Android akan meng-ekstrak dan meletakkannya di sistem (*root directory*) dengan struktur seperti ini:

```text
/data/adb/
├── modules/
│   └── zengobox/
│       ├── module.prop     # Identitas modul (Nama, Versi, Author) yang dibaca oleh aplikasi Magisk
│       ├── system.prop     # (Opsional) Injeksi sistem properties / build.prop
│       ├── service.sh      # Script Magisk (Berjalan otomatis saat HP booting/menyala)
│       ├── action.sh       # Script aksi manual (Tombol di aplikasi Magisk untuk menyalakan/mematikan proxy)
│       └── uninstall.sh    # Script pembersihan otomatis ketika Anda menghapus modul dari Magisk
│
└── zengobox/               # (Folder Kerja / Working Directory Utama ZenGoBox)
    ├── bin/
    │   └── zengobox        # Program utama Golang yang sudah dikompilasi (Executable Binary)
    │
    ├── config/             # Folder konfigurasi
    │   ├── zengobox.yaml   # Konfigurasi master ZenGoBox (pengaturan port, mode iptables, auto-update)
    │   └── clash/          # Folder konfigurasi inti proxy (Mihomo/Sing-box)
    │       ├── config.yaml # Aturan perutean (rules, proxies, proxy-groups) milik Mihomo/Clash
    │       ├── geoip.dat   # Database alamat IP negara/regional
    │       └── geosite.dat # Database daftar domain/website
    │
    ├── mihomo              # Binary inti dari proxy pihak ketiga (Bisa diganti dengan sing-box)
    ├── dashboard/          # Folder tempat ekstraksi Web UI pihak ketiga (Yacd, Zashboard)
    │
    └── run/                # Folder penyimpanan data sementara (Sementara HP hidup)
        ├── zengobox.pid    # Menyimpan ID Proses (PID) Daemon ZenGoBox agar tidak berjalan ganda
        ├── core.pid        # Menyimpan ID Proses dari proxy (Mihomo)
        └── runs.log        # File log rekaman pergerakan proxy (Bisa dibaca lewat Web UI atau cat runs.log)
```

### Penjelasan Folder Sistem:
- Folder `/data/adb/modules/zengobox/` diawasi ketat oleh **Magisk** untuk urusan *booting* dan *user interface* modul.
- Folder `/data/adb/zengobox/` adalah wilayah bebas milik ZenGoBox. Folder ini terpisah agar ketika Anda memperbarui modul di Magisk, konfigurasi dan database *geo* Anda yang sudah ada tidak terhapus (Persistent Storage).
