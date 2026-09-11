# ⚡ Fast Go + HTMX ToDo App

A simple, fast, and lightweight ToDo app built with **Go**, **HTMX**, and **bbolt**.

No bloated frameworks.  
No external database.  
Just a clean little app that does its job.

---

## 🛠️ Built With

- **Go** — Backend & HTTP server
- **HTMX** — Dynamic UI without a heavy JavaScript stack
- **bbolt** — Embedded key-value database
- **HTML / CSS** — Frontend

---

## ✨ Why bbolt?

Instead of running a separate SQL database, this app keeps everything inside a local `todos.db` file.

- 🪶 Lightweight and embedded
- 🚫 No database server required
- 🔐 ACID transactions
- ⚡ Fast local reads & writes
- 📦 Pure Go
- 💾 Simple local database file

---

## 🚀 Run Locally

Clone the repository and start the server:

```bash
go run main.go
