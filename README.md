# Pocketa — Production Peer-to-Peer Wallet & Neobank API

**Pocketa** is a high-performance, self-contained peer-to-peer digital wallet and neobanking web app designed for India. Built with Go standard library `net/http` and pure Go SQLite, Pocketa provides instant @handle transfers, automated bill splitting, money requests, direct bank withdrawals, and real-time ledger accounting.

---

## 🌟 Key Features

- **Instant @Handle Transfers**: Send money to any friend using custom handles like `@riya.s` without needing bank account numbers or IFSC codes.
- **Bill Splitting & Requests**: Request money from friends or split restaurant, trip, and utility bills evenly with automated remainder handling.
- **Direct Bank Withdrawals**: Withdraw wallet funds directly to any Indian bank (HDFC, ICICI, SBI, Axis, Kotak) with IMPS integration and flat ₹2.00 fee.
- **Double-Entry Ledger Accounting**: Every top-up, transfer, and payout is recorded as signed delta entries in an audit-ready transactions ledger.
- **Risk & Operations Console**: Built-in admin dashboard for real-time volume metrics, top sender monitoring, wallet freezing, and automated reconciliation checks.
- **Single Self-Contained Binary**: All web templates and static assets are embedded into the Go binary via `embed.FS`. Zero external dependencies or Node build steps required.

---

## 🏛️ System Architecture

```
                                  +-----------------------+
                                  |   Pocketa Client      |
                                  | (Vanilla JS + HTML)   |
                                  +-----------+-----------+
                                              |
                                              v
                              +---------------+---------------+
                              |    Go net/http Server         |
                              |  (Embedded Templates & JS)    |
                              +---------------+---------------+
                                              |
                 +----------------------------+----------------------------+
                 |                            |                            |
                 v                            v                            v
      +----------+----------+      +----------+----------+      +----------+----------+
      | Auth & Session Svc  |      | Wallet & Ledger Svc |      | Transfer & Split Svc|
      +----------+----------+      +----------+----------+      +----------+----------+
                 |                            |                            |
                 +----------------------------+----------------------------+
                                              |
                                              v
                                  +-----------+-----------+
                                  |    SQLite Database    |
                                  | (modernc.org/sqlite)  |
                                  +-----------------------+
```

---

## ⚙️ Configuration & Environment Variables

| Env Variable | Default Value | Description |
|---|---|---|
| `PORT` | `8080` | Server HTTP listening port |
| `HOST` | `0.0.0.0` | Bind address for incoming connections |
| `DATABASE_PATH` | `./data/pocketa.db` | SQLite database file path (parent directory created automatically) |
| `ADMIN_EMAIL` | `risk@pocketa.money` | Seeded risk/operations admin email |
| `ADMIN_PASSWORD` | `pocketa-admin` | Seeded risk/operations admin password |
| `SEED_DATA` | `true` | Idempotently seed users, wallets, and 90 days of history on startup |

---

## 🚀 Quick Start & Building

### Prerequisites

- **Go 1.22+** (Go 1.26 included)

### Build and Run

```bash
# Build binary
go build -o bin/pocketa ./cmd/pocketa

# Run binary
./bin/pocketa
```

Alternatively, run directly:

```bash
go run ./cmd/pocketa
```

Access the application at `http://localhost:8080`.

### Run Unit Tests

```bash
go test ./...
```

---

## 🔐 Seeded Accounts

| Role | Email | Password | Handle | Initial Balance |
|---|---|---|---|---|
| **User** | `riya@pocketa.money` | `riya-2026` | `@riya.s` | ₹24,500.00 |
| **User** | `arjun@pocketa.money` | `arjun-2026` | `@arjun.m` | ₹48,300.00 |
| **Admin** | `risk@pocketa.money` | `pocketa-admin` | `@risk.ops` | ₹1,00,000.00 |

---

## 💰 Fee Schedule & Limits

| Transaction Type | Fee | Daily Limit (Basic KYC) | Daily Limit (Full KYC) |
|---|---|---|---|
| Wallet Top-Up | **FREE (₹0.00)** | ₹10,000.00 | ₹1,00,000.00 |
| Peer-to-Peer Transfer | **FREE (₹0.00)** | ₹10,000.00 | ₹1,00,000.00 |
| Bill Split & Request | **FREE (₹0.00)** | ₹10,000.00 | ₹1,00,000.00 |
| Bank Withdrawal | **₹2.00 flat** | ₹10,000.00 | ₹1,00,000.00 |

---

## 📡 REST API Reference (`/api/v1`)

### Authentication
- `POST /api/v1/auth/signup` — Create user account & instant wallet
- `POST /api/v1/auth/login` — Authenticate user & issue bearer session token
- `POST /api/v1/auth/logout` — Revoke active session token
- `GET /api/v1/me` — Get authenticated user details & wallet

### Wallet & Ledger
- `GET /api/v1/wallet` — Get wallet balance, limits, and sent today volume
- `POST /api/v1/wallet/top-up` — Top up wallet balance via UPI, Card, or Netbanking
- `GET /api/v1/wallet/transactions` — List wallet transactions with filter & pagination
- `GET /api/v1/wallet/statement.csv` — Export transaction history as CSV statement

### Transfers & Requests
- `GET /api/v1/users/lookup?handle=` — Lookup recipient wallet by @handle
- `POST /api/v1/transfers` — Execute peer-to-peer money transfer
- `GET /api/v1/transfers` — List transfers for authenticated user
- `POST /api/v1/requests` — Create single money request
- `POST /api/v1/split` — Split bill among multiple @handles
- `GET /api/v1/requests?direction=incoming|outgoing` — List money requests
- `POST /api/v1/requests/{id}/pay` — Pay pending money request
- `POST /api/v1/requests/{id}/decline` — Decline pending request
- `POST /api/v1/requests/{id}/cancel` — Cancel pending request

### Bank Accounts & Withdrawals
- `GET /api/v1/bank-accounts` — List user bank accounts
- `POST /api/v1/bank-accounts` — Link new Indian bank account
- `DELETE /api/v1/bank-accounts/{id}` — Remove bank account
- `POST /api/v1/withdrawals` — Withdraw wallet balance to bank account
- `GET /api/v1/withdrawals` — List past bank withdrawals

### Admin Console (Admin Role Required)
- `GET /api/v1/admin/wallets` — List all registered wallets
- `GET /api/v1/admin/wallets/{id}` — Get wallet details with transaction history
- `POST /api/v1/admin/wallets/{id}/freeze` — Freeze wallet
- `POST /api/v1/admin/wallets/{id}/unfreeze` — Unfreeze wallet
- `GET /api/v1/admin/transfers` — List all system transfers
- `GET /api/v1/admin/metrics` — Aggregate volume, balances, and failure rate metrics
- `GET /api/v1/admin/ledger/check` — Execute double-entry ledger reconciliation check

---

## 📜 License

MIT License. Copyright (c) 2026 Pocketa Financial Technologies Pvt Ltd.
