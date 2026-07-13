# TSC Forex — Live Gold & FX Platform (Go backend + iOS app)

Production-grade Go backend that ingests real-time market data, computes a full
catalog of **gold products** (gram, çeyrek, yarım, tam, cumhuriyet, ata, 22/18/14
ayar, ons) and **currencies** (USD, EUR, GBP, JPY vs TRY), persists everything
for charting/AI, and streams updates to thousands of concurrent clients over
WebSocket. Users register/login, build a watchlist, set price-threshold alerts,
and receive real-time notifications. Ships with a **SwiftUI iOS app**.

```
GramGoldTRY   = (XAUUSD / 31.1034768) * USDTRY
ProductTRY    = GramGoldTRY * gramWeight * fineness   (çeyrek, tam, ata, …)
EURTRY/GBPTRY/JPYTRY = derived from the FX feed
```

Data refreshes every **5 seconds** (configurable) via the `synthetic` provider
(offline, always-on) or the `rest` provider (real APIs: open.er-api.com FX +
optional gold source). Providers are swappable with automatic failover.

## Architecture

Clean Architecture with a single-calculation, many-broadcast pipeline:

```
Providers → EventBus → Calculator → EventBus ─┬→ GoldBroadcaster → WS Hub → Clients
                                              ├→ CacheBridge → Redis (SET + PUB)
                                              └→ HistoryWriter (worker pool) → PostgreSQL

Redis PUB ⇄ SUB  ← multi-instance fan-in (skips own messages by instance id)
```

- **Domain** owns the formula and types (`internal/domain`).
- **Calculator** is a single writer (no mutex on the hot path); publishes only on material change.
- **Providers** are swappable behind `MarketProvider`; `ProviderManager` handles exponential-backoff reconnect and automatic failover across a provider chain.
- **WebSocket Hub** uses the classic hub pattern; slow clients get dropped (bounded per-client buffers) so 10k clients never block the hub.
- **Redis** caches the latest price and provides pub/sub so multiple backend instances stay in sync.
- **PostgreSQL** (GORM) persists every calculated price asynchronously and serves history + OHLC aggregation.
- **Redis/Postgres are optional at boot** — if unreachable the service degrades gracefully (mock provider + calculator + WS still serve).

## Tech Stack

Go 1.25 · Gin · gorilla/websocket · Viper · Zap · validator · google/uuid ·
PostgreSQL + GORM · Redis · Prometheus · JWT + bcrypt · Docker · Nginx.

## Project Layout

```
cmd/server            entrypoint, DI wiring, graceful shutdown
internal/
  domain              Tick, GramGold, formula, provider health
  provider            MarketProvider, Manager (reconnect/failover), mock + stubs
  calculator          single-writer gram-gold calculator
  websocket           Hub, Client, Handler (ping/pong, compression, drop policy)
  service             GoldBroadcaster, CacheBridge, HistoryWriter, AuthService
  cache               Redis latest-price cache + pub/sub
  repository          GORM gold + user repositories, OHLC aggregation
  database            GORM connection + migrations
  models              GORM models (gold_prices, users)
  api                 HTTP handlers + DTOs
  routes              Gin engine wiring + middleware
  middleware          request id, security headers, CORS, JWT, rate limit, metrics
  config              Viper typed config + hot reload
  logger              Zap structured logging
  metrics             Prometheus collectors
pkg/eventbus          in-process typed pub/sub (hot path)
docs                  OpenAPI spec + embedded Swagger UI
deploy                nginx, prometheus, grafana configs
tsc_docker            Dockerfile + docker-compose
tests                 integration + benchmarks
```

## API

Base path: `/api/v1`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/gold/current` | Latest gram gold price |
| GET | `/gold/history` | History; `?interval=1m,5m,15m,30m,1h,4h,1d` for OHLC |
| GET | `/instruments` | Full instrument catalog (metadata) |
| GET | `/instruments/prices` | Live prices for all gold products + currencies |
| GET | `/instruments/:code` | Latest price for one instrument |
| GET | `/instruments/:code/history` | Per-instrument history / OHLC |
| GET | `/providers` | Provider status + failover chain |
| GET | `/health` · `/livez` · `/readyz` | Health + k8s probes |
| POST | `/auth/register` | Register user (argon2id) |
| POST | `/auth/login` | Login → JWT access + rotating refresh |
| POST | `/auth/refresh` | Rotate refresh token |
| POST | `/auth/logout` | Revoke refresh token |
| GET/POST/DELETE | `/me/watchlist[/:code]` | Manage tracked instruments (JWT) |
| GET/POST/DELETE | `/me/alerts[/:id]` | Manage price-threshold alerts (JWT) |
| GET | `/me/notifications` | Notification history (JWT) |
| POST | `/me/devices` | Register APNs/FCM push token (JWT) |
| GET (WS) | `/ws?token=<jwt>` | Live price stream + targeted notifications |
| GET | `/metrics` | Prometheus metrics |
| GET | `/docs` | Swagger UI |

## Quick Start

### Local (no infra required — degraded mode)

```bash
make run
# WebSocket:
websocat ws://localhost:8080/ws
# REST:
curl localhost:8080/api/v1/gold/current
```

### Full stack (Docker)

```bash
make compose-up
# Nginx  → http://localhost:8080
# Prom   → http://localhost:9090
# Grafana→ http://localhost:3000  (anonymous enabled)
```

## Configuration

Precedence: defaults < `configs/config.yaml` < `.env` < environment variables.
Env vars use the `APP_` prefix with underscores, e.g. `server.port` → `APP_SERVER_PORT`.
See `.env.example`. The config file is hot-reloaded (invalid reloads are rejected,
keeping the previous snapshot).

## Testing

```bash
make test              # unit tests
make test-cover        # coverage report → coverage.html
make test-integration  # needs a reachable Postgres
make bench             # hot-path benchmarks
```

Benchmark highlights (Apple M3):

```
BenchmarkGramGoldFormula     0.30 ns/op    0 allocs
BenchmarkEventBusPublish    35.4  ns/op    1 alloc
BenchmarkCalculatorHotPath 274   ns/op     2 allocs
```

End-to-end update latency is well within the sub-100 ms target.

## Observability

Prometheus metrics include connected clients, dropped messages, broadcasts,
calculations, provider reconnects/switches, and Redis/DB latency & errors.
Structured Zap logs carry `timestamp`, `request_id`, `provider`, `symbol`,
`latency`, and `error`.

## Extensibility

The provider abstraction and symbol-agnostic pipeline are designed to add
Silver, EUR/TRY, USD Index, BTC/ETH, BIST, alerts, notifications, and
candlestick/technical indicators (RSI, MACD, MAs) without structural changes.

## Security & modern auth

- **argon2id** password hashing (memory-hard; OWASP-aligned) with transparent
  bcrypt-legacy verification for migration.
- **Rotating, revocable refresh tokens** stored hashed (SHA-256) in Postgres;
  every refresh revokes the previous token. `/auth/logout` revokes a session.
- JWT (HS256) access tokens, per-IP rate limiting, CORS allow-list, security
  headers, request validation, graceful shutdown. Restrict `/metrics` in prod
  and set a strong `APP_JWT_SECRET`.

## Multi-instrument, alerts & notifications

- Full catalog: 10 gold products + 4 currencies, all computed and persisted.
- Per-user **watchlist** and **price-threshold alerts** (above/below) with a
  configurable cooldown to avoid alert storms.
- **Alert engine** evaluates live updates off the hot path and fires
  **notifications** delivered instantly over the user's authenticated WebSocket
  channel, persisted in Postgres, and optionally pushed via **APNs**.

## AI-ready storage (pgvector)

Postgres runs the `pgvector/pgvector:pg16` image. Migrations enable the `vector`
extension and create a `price_embeddings` table, reserved for future AI
(pattern similarity, forecasting, anomaly detection).

## Kubernetes

`deploy/k8s` contains a full kustomize bundle: namespace, ConfigMap/Secret,
Postgres StatefulSet (pgvector), Redis, the app Deployment with `/livez` +
`/readyz` probes, a **HorizontalPodAutoscaler** (3→20 pods on CPU/memory), a
`ClusterIP` Service, and an nginx Ingress with WebSocket-friendly timeouts.

```bash
kubectl apply -k deploy/k8s
```

## iOS app (SwiftUI)

`ios/tscforex` is a SwiftUI app: login/register, live prices for all instruments
(via WebSocket), per-instrument detail with alarm creation, alerts management,
and notifications. Set your Mac's LAN IP in the app's **Settings** tab (or edit
`AppConfig.swift`) so the iPhone reaches the backend over Wi-Fi.

> Requires full **Xcode** (not just Command Line Tools) to build and run on a
> device. Open `ios/tscforex/tscforex.xcodeproj`, select your iPhone, set your
> signing team, and Run. ATS is pre-configured for local-network HTTP/WS.
