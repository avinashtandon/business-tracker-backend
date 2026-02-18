---
description: How to set up and run a Go REST API from scratch
---

# Go REST API — Step-by-Step Setup

## 1. Initialize the Project

```bash
mkdir my-app && cd my-app
go mod init github.com/<your-username>/<your-app-name>
```

Creates `go.mod` — the dependency manifest for your project.

---

## 2. Set Up Project Structure

```
my-app/
├── cmd/
│   └── api/
│       └── main.go        # Entry point
├── internal/
│   ├── config/            # App config (env vars)
│   ├── handler/           # HTTP handlers
│   ├── middleware/        # Auth, logging, rate limiting
│   ├── model/             # Data models / structs
│   ├── repository/        # DB queries
│   └── service/           # Business logic
├── migrations/            # SQL migration files
├── .env                   # Local environment variables (never commit)
├── .env.example           # Template for .env (commit this)
├── go.mod
└── go.sum
```

---

## 3. Add Common Dependencies

```bash
# HTTP router
go get github.com/go-chi/chi/v5

# MySQL driver + sqlx
go get github.com/go-sql-driver/mysql
go get github.com/jmoiron/sqlx

# JWT auth
go get github.com/golang-jwt/jwt/v5

# Load .env files
go get github.com/joho/godotenv

# Input validation
go get github.com/go-playground/validator/v10

# DB migrations
go get github.com/golang-migrate/migrate/v4

# UUIDs
go get github.com/google/uuid

# Password hashing (stdlib, no install needed)
# golang.org/x/crypto
go get golang.org/x/crypto

# Rate limiting (stdlib, no install needed)
go get golang.org/x/time
```

---

## 4. Create `.env` File

```env
APP_ENV=development
PORT=8080
DB_DSN=user:password@tcp(localhost:3306)/dbname?parseTime=true
JWT_SECRET=your-secret-key
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d
```

> Never commit `.env`. Add it to `.gitignore`.

---

## 5. Write `main.go`

Key things `main.go` should do:
1. Load `.env` with `godotenv.Load()`
2. Load config from environment variables
3. Connect to the database
4. Run migrations
5. Set up router and middleware
6. Start HTTP server
7. Handle graceful shutdown on `SIGINT`/`SIGTERM`

---

## 6. Write SQL Migrations

Create numbered migration files in `migrations/`:

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_sessions_table.up.sql
└── 000002_create_sessions_table.down.sql
```

---

## 7. Sync Dependencies

After adding all imports to your code:

```bash
go mod tidy
```

This downloads all missing packages and cleans up unused ones.

---

## 8. Run the Server

```bash
go run ./cmd/api/main.go
```

---

## 9. If Port Is Already in Use

```bash
lsof -ti :8080 | xargs kill -9
```

Then re-run the server.

---

## 10. Stop the Server

Press `Ctrl+C` — the server should shut down gracefully.

---

## 11. Build for Production

```bash
go build -o bin/api ./cmd/api/main.go
./bin/api
```

---

## Common Troubleshooting

| Error | Fix |
|---|---|
| `could not import <package>` | Run `go mod tidy` |
| `address already in use` | Kill the process on that port (Step 9) |
| `no such file .env` | Create your `.env` from `.env.example` |
| `migrate: no migration` | Check your `migrations/` folder path in config |
