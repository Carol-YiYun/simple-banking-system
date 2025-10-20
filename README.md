# 🏦 Simple Banking System (Golang)

  

A lightweight **RESTful Banking API** built with Go, featuring:

- In-memory data persistence (with JSON snapshot)

- Atomic transactions

- Full unit & integration tests

- Docker-ready deployment
  
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

Q2/

├── cmd/

│ └── server/ # Entry point (main.go)

├── internal/

│ ├── bank/ # Core business logic

│ ├── server/ # RESTful API layer

│ └── storage/ # JSON snapshot persistence

├── Dockerfile

├── go.mod / go.sum

└── README.md


---

## ⚙️ Quick Start

### 1️⃣ Run Tests (Unit + Integration)

```bash

go mod tidy

go test ./... -race -v
```

✅ Expected result:
```bash
=== RUN TestHTTPFlowAndPersistHook

--- PASS: TestHTTPFlowAndPersistHook (0.12s)

=== RUN TestConcurrentTransfersAtomicity

--- PASS: TestConcurrentTransfersAtomicity (0.01s)

PASS

ok banking/internal/bank 0.3s

ok banking/internal/server 0.5s

ok banking/internal/storage 0.1s
```
  
2️⃣ Run the Server

```bash

go run ./cmd/server

```

Server starts at http://localhost:8080.
  

Check health:

```bash
curl localhost:8080/health
# → "ok"
```


3️⃣ Example API Flow

```bash

# Create two accounts

curl -X POST localhost:8080/accounts \

-H "Content-Type: application/json" \

-d '{"name":"Alice","balance":1000}'

  

curl -X POST localhost:8080/accounts \

-H "Content-Type: application/json" \

-d '{"name":"Bob","balance":500}'

  

# Deposit to Alice

curl -X POST localhost:8080/accounts/<alice_id>/deposit \

-H "Content-Type: application/json" \

-d '{"amount":200}'

  

# Withdraw from Bob

curl -X POST localhost:8080/accounts/<bob_id>/withdraw \

-H "Content-Type: application/json" \

-d '{"amount":100}'

  

# Transfer

curl -X POST localhost:8080/transfer \

-H "Content-Type: application/json" \

-d '{"From":"<alice_id>","To":"<bob_id>","Amount":300}'

  

# View logs

curl localhost:8080/accounts/<bob_id>/logs

```

  

🧪 Test Coverage Summary
|**Requirement**|**Status**|**Test File**|
|---|---|---|
|RESTful API endpoints|✅|internal/server/server_test.go|
|Account balance cannot be negative|✅|TestCreateNegativeBalance|
|Create account|✅|TestCreateAndListGet|
|Deposit / Withdraw|✅|TestDepositWithdraw|
|Transfer (atomic)|✅|TestTransfer, TestConcurrentTransfersAtomicity|
|Transaction logs (when, amount, target)|✅|TestLogs|
|Atomic transaction|✅|TestConcurrentTransfersAtomicity|
|Unit tests & Integration tests|✅|All _test.go files|
|Docker container run server|✅|Dockerfile|


### 🚀 Quick Run from Docker Hub (for reviewers)

If you only want to verify the running service without building locally,  
just pull and run the image from Docker Hub:

```bash
docker run -p 8080:8080 carollin/q2-banking:latest
curl localhost:8080/health
```

## **🐳 Run with Docker**

Build and run the server:
```bash
docker build -t banking .
docker run -p 8080:8080 banking
```

Verify:
```bash
curl localhost:8080/health
```

## **🧠 Technical Highlights**

- **Layered architecture** (bank, server, storage)
    
- **Atomic transactions** with mutex for concurrent safety
    
- **Comprehensive tests** covering logic, HTTP, and persistence
    
- **JSON snapshot persistence**, easily replaceable with SQLite or Redis
    
- **Clean, idiomatic Go** with no external dependencies beyond stdlib
    

---

## **✨ Quick Demo (60 seconds)**
```bash
docker run -p 8080:8080 <your-dockerhub-username>/banking
curl localhost:8080/health
```
✅ Service ready. All requirements verified by automated tests.


### **Author**

  

**Carol Lin (YiYun Lin)**

Backend Engineer — Go

GitHub: [Carol-YiYun](https://github.com/Carol-YiYun)
