Tugas Besar Strategi Algoritma — Tubes-2-Stima
#Deskripsi
Program ini merupakan aplikasi pencari resep dari elemen pada permainan Little Alchemy 2.

#Algoritma
Algoritma pencarian yang diimplementasikan adalah Depth First Search (DFS), Breadth First Search (BFS), dan Bidirectional Search:
- DFS adalah pencarian secara mendalam kepada salah satu elemen pembentuk. Jadi, pencarian dilakukan untuk menemukan pembentuk (elemen dasar) dari salah satu elemen pembentuk awal sebelum berpindah ke elemen selanjutnya.

BFS bekerja secara melebar dengan mencari seluruh elemen pembentuk di setiap tingkatan secara bersamaan.

Bidirectional merupakan pencarian dengan 2 arah berbeda secara bersamaan. Akan ada pencarian maju dan mundur yang bergerak dari sumber yang berbeda dengan arah yang berbeda. Ketika elemen yang dicek sama, maka hasil akan diberikan.

🛠️ Requirement
Instalasi Node.js

Instalasi Docker

Instalasi bahasa Go

🧪 Command
💻 Menggunakan Docker:
Cukup jalankan perintah berikut:

docker compose up
⚙️ Tanpa Docker:
Jalankan Backend
Pindah ke direktori backend lalu jalankan:

bash
Copy
Edit
go run .
Jalankan Frontend
Buka terminal baru, pindah ke direktori frontend lalu jalankan:

bash
Copy
Edit
npm install
npm run dev
👨‍💻 Author
Alvin Christopher Santausa  13523033

Kenneth Poenadi       13523040

Ivan Wirawan         13523046

