<img src="https://cdn.slicesoft.dev/boat.svg" width="400" />

# Keel Addon Template
Keel is a Go framework for building REST APIs with modular 
architecture, automatic OpenAPI, and built-in validation.

[![CI](https://github.com/slice-soft/ss-keel-core/actions/workflows/ci.yml/badge.svg)](https://github.com/slice-soft/ss-keel-core/actions)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
[![Go Report Card](https://goreportcard.com/badge/github.com/slice-soft/ss-keel-core)](https://goreportcard.com/report/github.com/slice-soft/ss-keel-core)
[![Go Reference](https://pkg.go.dev/badge/github.com/slice-soft/ss-keel-core.svg)](https://pkg.go.dev/github.com/slice-soft/ss-keel-core)
![License](https://img.shields.io/badge/License-MIT-green)
![Made in Colombia](https://img.shields.io/badge/Made%20in-Colombia-FCD116?labelColor=003893)


## Database addon for Keel

`ss-keel-gorm` adds SQL database support to a [Keel](https://keel-go.dev) project via [GORM](https://gorm.io).
It is the official addon for relational databases in the Keel ecosystem.

---

## 🚀 Installation

```bash
keel add gorm
```

The Keel CLI will:
1. Add `github.com/slice-soft/ss-keel-gorm` as a dependency.
2. Import the `database` package in `cmd/main.go` and inject initialization code.
3. Add a `DATABASE_URL` environment variable example to your `.env`.

---

## ⚙️ Configuration

### Via DSN (recommended)

```go
db, err := database.New(database.Config{
  Engine: database.EnginePostgres,
  DSN:    config.GetEnvOrDefault("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable"),
  Logger: app.Logger(),
})
if err != nil {
  app.Logger().Error("failed to start app: %v", err)
}
defer db.Close()
```

### Via individual fields

```go
db, err := database.New(database.Config{
  Engine:   database.EnginePostgres,
  Host:     "localhost",
  Port:     5432,
  User:     "postgres",
  Password: "postgres",
  Database: "app",
})
```

---

## 🗄️ Supported engines

| Engine | Constant |
|---|---|
| PostgreSQL | `database.EnginePostgres` |
| MySQL | `database.EngineMySQL` |
| MariaDB | `database.EngineMariaDB` |
| SQLite | `database.EngineSQLite` |
| SQL Server | `database.EngineSQLServer` |

---

## 🔗 Connection pool

Pool defaults applied when not overridden:

| Parameter | Default |
|---|---|
| `MaxOpenConns` | 25 |
| `MaxIdleConns` | 5 |
| `ConnMaxLifetime` | 30 min |
| `ConnMaxIdleTime` | 15 min |

Override via `Config.Pool`:

```go
db, err := database.New(database.Config{
  Engine: database.EnginePostgres,
  DSN:    config.GetEnvOrDefault("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable"),
  Logger: app.Logger(),
  Pool: database.PoolConfig{
    MaxOpenConns:    50,
    MaxIdleConns:    10,
    ConnMaxLifetime: time.Hour,
    ConnMaxIdleTime: 20 * time.Minute,
  },
})
```

---

## 📦 Generic repository

`GormRepository[T, ID]` implements `core.Repository[T, ID]` and provides standard CRUD out of the box.

```go
type UserRepository = database.GormRepository[User, string]

func NewUserRepository(db *database.DBinstance) *UserRepository {
    return database.NewGormRepository[User, string](db)
}
```

Available methods: `FindByID`, `FindAll` (paginated), `Create`, `Update`, `Delete`.
Use `DB()` to access the underlying `*gorm.DB` for custom queries.

---

## ❤️ Health checker

Register the database in the Keel health endpoint:

```go
app.RegisterHealthChecker(database.NewHealthChecker(db))
```

This exposes the database status under `GET /health`:

```json
{ "database": "UP" }
```

---

## 🔌 Custom engines (Oracle and others)

Use `RegisterDialector` to plug any third-party dialector:

```go
_ = database.RegisterDialector(database.EngineOracle, func(cfg database.Config) (gorm.Dialector, error) {
  // return oracleDriver.Open(cfg.DSN), nil
  return nil, errors.New("wire your Oracle dialector here")
})
```

---

## 🤚 CI/CD and releases

- **CI** runs on every pull request targeting `main` via `.github/workflows/ci.yml`.
- **Releases** are created automatically on merge to `main` via `.github/workflows/release.yml` using Release Please.

---

## 💡 Recommendations

* Use `DSN` for production deployments; it keeps the config flexible and avoids exposing individual credentials in code.
* Embed `GormRepository` or alias it in your domain layer to keep infrastructure details out of your business logic.
* Register `NewHealthChecker` so Keel's `/health` endpoint always reflects real database connectivity.

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for setup and repository-specific rules.
The base workflow, commit conventions, and community standards live in [ss-community](https://github.com/slice-soft/ss-community/blob/main/CONTRIBUTING.md).

## Community

| Document | |
|---|---|
| [CONTRIBUTING.md](https://github.com/slice-soft/ss-community/blob/main/CONTRIBUTING.md) | Workflow, commit conventions, and PR guidelines |
| [GOVERNANCE.md](https://github.com/slice-soft/ss-community/blob/main/GOVERNANCE.md) | Decision-making, roles, and release process |
| [CODE_OF_CONDUCT.md](https://github.com/slice-soft/ss-community/blob/main/CODE_OF_CONDUCT.md) | Community standards |
| [VERSIONING.md](https://github.com/slice-soft/ss-community/blob/main/VERSIONING.md) | SemVer policy and breaking changes |
| [SECURITY.md](https://github.com/slice-soft/ss-community/blob/main/SECURITY.md) | How to report vulnerabilities |
| [MAINTAINERS.md](https://github.com/slice-soft/ss-community/blob/main/MAINTAINERS.md) | Active maintainers |

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- Website: [keel-go.dev](https://keel-go.dev)
- GitHub: [github.com/slice-soft/ss-keel-cli](https://github.com/slice-soft/ss-keel-cli)
- Documentation: [docs.keel-go.dev](https://docs.keel-go.dev)

---

Made by [SliceSoft](https://slicesoft.dev) — Colombia 💙
