# Fast Go + bbolt + HTMX ToDo App

A lightning-fast, zero-dependency ToDo web application powered by **Go**, **HTMX**, and **bbolt** (embedded pure-Go key-value NoSQL database).

---

## Why bbolt instead of SQL?

- **Zero Database Server Setup**: No MySQL or PostgreSQL daemon required. No username, password, or connection string fumbling.
- **Embedded & Pure Go**: The database compiles directly into the Go binary. Data is safely stored in a local `todos.db` file.
- **Microsecond Speed**: bbolt uses memory-mapped I/O (`mmap`) with ACID transactions, making data retrieval virtually instantaneous.
- **Cross-Device Live Sync**: Access the exact same live list from your desktop and your phone at the same time.

---

## How to Run

### 1. Run Locally
```powershell
go run main.go
```
The server will start on `http://0.0.0.0:8080` and display your local network URL:
```
==================================================
🚀 ToDo App running with fast embedded bbolt database!
💻 Desktop Access : http://localhost:8080
📱 Phone Access   : http://192.168.1.XX:8080 (on same Wi-Fi)
==================================================
```

### 2. Using from your Phone
1. Connect your phone to the **same Wi-Fi network** as your computer.
2. Open your phone browser (Safari, Chrome, etc.).
3. Navigate to the phone URL shown in the terminal (e.g. `http://192.168.1.XX:8080`).
4. Any tasks you add, complete, or delete on your phone or desktop will be instantly in sync.

---

## Deployment (Free Cloud Hosting)

To keep this website running 24/7 on the internet so you can use it anywhere (even outside your home Wi-Fi), you can deploy it to **Render**, **Fly.io**, or **Railway**:

1. Push this repository to GitHub.
2. Connect the repo on [Render](https://render.com) or [Fly.io](https://fly.io).
3. Set the build command to `go build -o server .` and start command to `./server`.
4. (Optional) On Fly.io / Railway, attach a persistent volume to the folder containing `todos.db`.
