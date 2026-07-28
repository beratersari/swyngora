# Watchlist store (`internal/adapter/watchliststore`)

Infrastructure adapters for `domain.WatchlistPort`.

## Implementations

| Type | Constructor | Persistence |
|------|-------------|-------------|
| **SQLite** (default in `cmd/server`) | `OpenSQLite(path)` | File-backed; survives restarts |
| **Memory** | `NewMemory()` | Process-only (tests / ephemeral) |

## Schema (SQLite)

- `watchlist_meta` — one row per `client_id` + `updated_at`
- `watchlist_items` — `(client_id, exchange, symbol)` primary key, note, added_at

Enforces `domain.MaxWatchlistItems` and `DefaultMaxClients` under a write lock.

## Config

| Env | Default |
|-----|---------|
| `WATCHLIST_DB_PATH` | `data/watchlist.db` |

## Tests

```bash
go test ./internal/adapter/watchliststore/ -count=1
```

`TestSQLite_PersistsAcrossReopen` closes the DB handle and reopens the same file to verify restart survival.