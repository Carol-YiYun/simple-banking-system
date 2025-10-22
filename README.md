# 🏦 Simple Banking System (Golang)

  

A lightweight **RESTful Banking API** built with Go, featuring:

- In-memory data persistence (with JSON snapshot)

- Atomic transactions

- Full unit & integration tests

- Docker-ready deployment
  
---

## 🌐 Live Demo (Deployed on Render)

This service is live on **Render.com** — no setup needed.

**Base URL:**  
👉 https://simple-banking-system-jh5f.onrender.com

**Quick Test:**
```bash
# Health check
curl https://simple-banking-system-jh5f.onrender.com/health
# → {"status":"ok"}
```
⚠️ The first request may take up to **1** minute as Render wakes from idle.
💡 You can check **/health** first to confirm readiness.

---

## 📘 Overview

Implements a simple banking system that supports:

| Feature | Description |
|----------|--------------|
| 🧾 Create Account | Create an account with name and balance (cannot be negative) |
| 💰 Deposit & Withdraw | Update balance safely with validation |
| 🔁 Transfer | Transfer money atomically between accounts |
| 📜 Transaction Logs | Records date/time, amount, direction, and counterparty |
| 🧩 Atomicity | Prevents race conditions in concurrent transactions |
| ✅ Full Testing | Includes unit & integration tests |
| 🐳 Docker | Run the whole service in a container |

---

## 🧱 Project Structure
```
SIMPLE-BANKING-SYSTEM/
├── cmd/
│ └── server/ # Entry point (main.go)
├── internal/
│ ├── bank/ # Core business logic
│ ├── server/ # RESTful API layer
│ └── storage/ # JSON snapshot persistence
├── Dockerfile
├── go.mod / go.sum
└── README.md
```

---

## ⚙️ Quick Run

You can verify the project in 3 ways:

1️⃣ Online via **Render.com**

2️⃣ Using the prebuilt **Docker image**

3️⃣ Running **locally** with Go

### 🌐 Option A. Test Online (Render.com)

Base URL:
🔗 https://simple-banking-system-jh5f.onrender.com

Health Check:
```bash
curl https://simple-banking-system-jh5f.onrender.com/health
# → {"status":"ok"}
```
⚠️ The first request may take up to one minute as the Render server wakes from idle.
💡 You can use **/health** first to confirm the API is ready.


### 🐳 Option B. Run via Docker

1️⃣ Pull and Run from Docker Hub
```bash
docker run --rm -p 8080:8080 docker.io/carollin/simple-banking-system:latest
```
2️⃣ Verify
```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```
3️⃣ (Optional) Build Locally
```bash
docker build -t simple-banking-system .
docker run --rm -p 8080:8080 simple-banking-system
```

### 💻 Option C. Run Locally (No Docker)

⚠️ Requires Go ≥ 1.22

1️⃣ Run tests (unit + integration) under project root
```bash
go mod tidy
go test ./... -race -v
```
2️⃣ Run server
```bash
go run ./cmd/server
```
3️⃣ Check health
```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```
---

## 📡 API Endpoints

| Method | Endpoint | Description |
|:-------|:----------|:------------|
| **GET** | `/health` | Check service status (`{"status":"ok"}`) |
| **POST** | `/accounts` | Create new account (`{"name":"Alice","balance":1000}`) |
| **GET** | `/accounts` | List all accounts |
| **GET** | `/accounts/{id}` | Retrieve single account details |
| **POST** | `/accounts/{id}/deposit` | Deposit funds (`{"amount":200}`) |
| **POST** | `/accounts/{id}/withdraw` | Withdraw funds (`{"amount":100}`) |
| **POST** | `/transfer` | Transfer between accounts (`{"From":"<id>","To":"<id>","Amount":300}`) |
| **GET** | `/accounts/{id}/logs` | View account transaction logs |

---

## 🧩 Suggested API Test Flow

Below is a quick example sequence to verify core features once the server is running:

```bash
# 1. Create two accounts
curl -X POST localhost:8080/accounts \
-H "Content-Type: application/json" \
-d '{"name":"Alice","balance":1000}'

curl -X POST localhost:8080/accounts \
-H "Content-Type: application/json" \
-d '{"name":"Bob","balance":500}'

# 2. Deposit to Alice
curl -X POST localhost:8080/accounts/<alice_id>/deposit \
-H "Content-Type: application/json" \
-d '{"amount":200}'

# 3. Withdraw from Bob
curl -X POST localhost:8080/accounts/<bob_id>/withdraw \
-H "Content-Type: application/json" \
-d '{"amount":100}'

# 4. Transfer from Alice to Bob
curl -X POST localhost:8080/transfer \
-H "Content-Type: application/json" \
-d '{"From":"<alice_id>","To":"<bob_id>","Amount":300}'

# 5. Check Bob's transaction logs
curl localhost:8080/accounts/<bob_id>/logs
```

💡 Tip: Replace **<alice_id>** and **<bob_id>** with the actual IDs returned when you create the accounts.

---

## 🧠 Technical Highlights

- **Layered Architecture** — clear separation of `bank` (business logic), `server` (HTTP API), and `storage` (persistence).  
- **Atomic Transactions** — concurrent-safe transfers implemented using mutex locks.  
- **Data Persistence** — in-memory state with JSON snapshot, easily replaceable with SQLite, Redis, or cloud storage.  
- **Comprehensive Testing** — full unit and integration coverage validated via `go test -race -v`.  
- **Stateless RESTful API** — clean endpoint design following REST principles.  
- **Dockerized Deployment** — fully containerized for consistent CI/CD and Render deployment.  
- **Zero External Dependencies** — uses only Go’s standard library for maximum portability.  

---

### **Author**
  

**Carol Lin (YiYun Lin)**

Backend Engineer — Go

GitHub: [Carol-YiYun](https://github.com/Carol-YiYun)
