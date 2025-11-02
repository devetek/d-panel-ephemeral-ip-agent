# Tukiran dan Marijan

Tukiran adalah `tunnel-client` yang membantu membuka local port di dalam jaringan private ke public melalui `dPanel Tunnel`. Sedangkan Marijan adalah `tunnel-client manager` yang membantu mengatur banyak port yang dapat terhubung ke tunnel melalui Tukiran. Keduanya merupakan bagian dari [dPanel](https://cloud.terpusat.com/), digunakan untuk membantu kamu agar dapat mengatur mesin melalui dPanel meskipun tidak memiliki static public IP.

Untuk dapat menggunakan Tukiran dan Marijan, kamu perlu melakukan instalasi `dPanel Agent` di sebuah server di dalam jaringan kamu. Dengan mengikuti langkah-langkah instalasi berikut ini:

1. Login ke [dPanel](https://cloud.terpusat.com/).
2. Masuk ke halaman profile kamu.
3. Buat private token, kemudian salin private token yang sudah dihasilkan.
4. Install `dPanel Agent` di sebuah server kamu dengan menjalankan perintah berikut ini:
```sh
curl -sSL https://artifact.dnocs.io/install-with-tunnel.sh | sh -s -- --private-token <PRIVATE-TOKEN>
```
Ganti `<PRIVATE-TOKEN>` dengan private token yang sudah kamu salin di langkah sebelumnya. Setelah instalasi selesai, kamu dapat mengatur untuk membuka port di dalam jaringan private melalui [dPanel Tunnel Manager](https://cloud.terpusat.com/tunnel).

5. Setelah membuat tunnel di dPanel. Kamu dapat mendaftarkan mesin yang terhubung tersebut di menu `Machines` di dPanel.

### Cara Beroperasi

`dPanel Agent` akan beroperasi secara otomatis setelah diinstal. Kamu dapat memeriksa status dari `dPanel Agent` dengan menjalankan perintah berikut ini:

#### Linux
```sh
systemctl status dpanel-agent
```

#### macOS
```sh
brew services list | grep dpanel-agent
```

Tukiran dan Marijan adalah paket yang sudah termasuk di dalam `dPanel Agent`. Kamu dapat menggunakannya setelah `dPanel Agent` beroperasi.

### Menggunakan Tukiran dan Marijan

Kamu dapat menggunakan Tukiran dan Marijan terpisah dari `dPanel Agent`. Jika hanya ingin mengekspose port di dalam jaringan private ke public, tanpa perlu mengatur mesin di dPanel. Ada 2 cara yang dapat digunakan:

1. Menggunakan binary yang sudah disediakan di [halaman release](https://github.com/devetek/tuman/releases). Download binary yang sesuai dengan platformmu. Kemudian, extract binary tersebut, dan jalankan binary tersebut dengan perintah berikut ini:
```sh
./marijan run --config <CONFIG-FILE>
```
Ganti `<CONFIG-FILE>` dengan path ke file config yang kamu miliki. Jika config tidak diatur, Marijan akan menggunakan config default di `~/.marijan/config.json`. File config memiliki format JSON, dan memiliki struktur sebagai berikut:

```json
[
  {
    "id": "tunnel-1",
    "tunnel_host": "tunnel.beta.devetek.app",
    "tunnel_port": "2220",
    "listener_host": "0.0.0.0",
    "listener_port": "3001",
    "service_host": "localhost",
    "service_port": "3000",
    "state": "active"
  }
]
```
Contoh config di atas akan membuat tunnel dengan id `tunnel-1` dengan state `active` yang artinya tunnel akan diaktifkan. Tunnel tersebut akan menghubungkan port `3001` di listener host ke port `3000` di lokal kamu. Tunnel akan terhubung ke `tunnel.beta.devetek.app` di port `2220`.

Contoh pengaturan dapat ditemukan di [files/config.json](files/config.json).

2. Menggunakan Tukiran dan Marijan sebagai library di dalam aplikasi kamu. Kamu dapat mengintegrasikan Tukiran dan Marijan ke dalam aplikasi kamu dengan menggunakan library yang disediakan.

```go
import (
	"github.com/devetek/tuman/pkg/marijan"
)

manager := marijan.NewManager(
    marijan.WithSource(marijan.ConfigSourceFile),
    marijan.WithURL("./config.json"),
    marijan.WithInterval(5*time.Second),
    marijan.WithDebug(true),
)

manager.Start()
```

Lihat di file [cmd/main.go](cmd/main.go) untuk cara penggunaan.

Kamu dapat menggunakan beberapa tunnel server berikut ini:
- `tunnel.beta.devetek.app`

Atau kamu dapat membuat tunnel server sendiri dengan repository [tunnel-server](https://github.com/devetek/tunnel-server). Dan deploy tunnel server di server kamu.