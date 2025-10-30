## Ephemeral IP Agent

Repository ini adalah simulasi bagaimana dPanel dapat berkomunikasi dengan mesin yang berada di dalam jaringan private yang tidak memiliki public IP. Contoh penggunaan metode ini salah satunya adalah ketika ingin mengakses Edge Computer (Raspberry Pi / Orange Pi) yang terdapat lokasi client yang tidak memiliki dedicated public IP.

Pertama-tama, edge computer yang akan digunakan harus di install `dPanel Ephemeral IP Agent` sebelum dibawa ke lokasi. Agen ini bertugas untuk menjalin komunikasi dengan `dPanel Manager` secara asyncronous melalui `dPanel Broker Message`. Kemudian jika `dPanel Manager` ingin menjalin komunikasi dengan `Edge Computer`, `dPanel Manager` akan mengirimkan permintaan melalui `dPanel Broker Message` untuk membuka jalur komunikasi di `Edge Computer` melalui `dPanel Public Tunnel`.

Setelah komunikasi terjalin antara `dPanel Ephemeral IP Agent` dan `dPanel Public Tunnel`, `dPanel Manager` dapat mengakses Edge Computer tersebut melalui `dPanel Public Tunnel`. Untuk selanjutnya menjalankan perintah ke dalam edge computer.

### Development

1. Jalankan `public-tunnel` dengan perintah `go run public-tunnel/main.go`
2. Jalankan `ephemeral-ssh-server` dengan perintah `go run private-server/main.go`
3. Jalankan `ephemeral-tunnel-client` dengan perintah `go run client-tunnel/*.go`
4. Akses `Edge Computer` dari `dPanel IaC` dengan perintah `ssh localhost -p 2221`, dan `http://localhost:3001` dari `dPanel Manager`

Berdasarkan 4 langkah yang dijalankan. Poin nomor 2 dan 3 adalah proses yang berjalan di dalam `Edge Computer`. Dan diatur oleh `dPanel Ephemeral IP Agent`. Sedangkan point nomor 1 adalah proses yang berjalan di dPanel Server yang diatur oleh `dPanel Manager`. Selanjutnya poin nomor 4 adalah proses yang dijalankan dPanel IaC untuk mengakses `Edge Computer` dan menjalankan perintah yang kita inginkan.

> [!TODO]
> 1. Memilih port tunnel secara acak sesuai dengan ketersediaan di `dPanel Public Tunnel`.

### Architecture

![Architecture](assets/architecture.jpg)

### Appendix
![Simulation](assets/simulation.png)
