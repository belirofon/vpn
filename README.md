<div align="center">
  <img src="https://img.shields.io/badge/status-active-success?style=flat-square" alt="Status" />
  <img src="https://img.shields.io/github/v/release/belirofon/vpn?style=flat-square" alt="Release" />
  <img src="https://img.shields.io/github/license/belirofon/vpn?style=flat-square" alt="License" />
  <img src="https://img.shields.io/github/last-commit/belirofon/vpn?style=flat-square" alt="Last Commit" />
  <img src="https://img.shields.io/badge/go-%3E%3D1.23-blue?style=flat-square&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/flutter-3.x-blue?style=flat-square&logo=flutter" alt="Flutter" />
</div>

<br />

<div align="center">
  <h1>🛡️ VPN Server & Client</h1>
  <p>
    <strong>Auto-proxy subscription fetcher · latency tester · geo-filter · one-tap mobile client</strong>
  </p>
  <p>
    <a href="#-features">Features</a> ·
    <a href="#-architecture">Architecture</a> ·
    <a href="#-quick-start">Quick Start</a> ·
    <a href="#-api-reference">API</a> ·
    <a href="#-mobile-client">Mobile Client</a> ·
    <a href="#-deploy">Deploy</a>
  </p>
</div>

---

## ✨ Features

| Capability | Description |
|---|---|
| **Subscription Fetcher** | Fetches proxy configs from your subscription URL (base64, JSON, or plain text) |
| **Protocol Parser** | Parses **VLESS**, **VMess**, **Trojan**, and **Shadowsocks** links |
| **Connectivity Tester** | Pings each server with real protocol handshakes (TCP, TLS, WebSocket, VLESS proxy) |
| **Geo-Filter** | Prefers non-Russian servers with fallback to all configs |
| **Auto-Select Best** | Returns the lowest-latency working config |
| **Auto-HTTPS** | Caddy reverse proxy with Let's Encrypt certificates |
| **DuckDNS** | Auto-updates DNS record for your domain |
| **Flutter Client** | Android & iOS app with one-tap connect/disconnect |
| **Refresh API** | Trigger config cache refresh on demand |
| **Periodic Updates** | Auto-refreshes config cache every `N` minutes |
| **REALITY Filter** | Skips VLESS+REALITY configs (not yet supported by Flutter client) |
| **Cloudflare WARP** | Generates WARP WireGuard config as a fallback tunnel |
| **Admin Panel** | Web-based admin: config management, WARP control, server monitoring |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Internet                           │
└────────┬────────────────────────────┬────────────────┘
         │                            │
    ┌────▼────┐                 ┌─────▼─────┐
    │  Port 80│                 │ Port 8443 │
    │ (HTTP)  │                 │ (HTTPS)   │
    └────┬────┘                 └─────┬─────┘
         │                            │
         └──────────┬─────────────────┘
                    │
           ┌────────▼────────┐
           │      Caddy       │  ← Auto HTTPS (Let's Encrypt)
           │  Reverse Proxy   │
           └────────┬────────┘
                    │
           ┌────────▼────────┐
           │   Go Server      │  ← Port :8080 (internal)
           │   (Gin Router)   │
           └────────┬────────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
     ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│ DuckDNS  │  │ GeoIP DB  │  │ Subscription│
│ (auto IP)│  │(MaxMind)  │  │ URL        │
└──────────┘  └──────────┘  └──────┬───┘
                                   │
                           ┌───────▼───────┐
                           │   Proxy Configs│
                           │ VLESS / VMess  │
                           │ Trojan / SS    │
                           └───────────────┘
```

### Component Diagram

| Component | Role |
|---|---|
| **Caddy** | Reverse proxy, TLS termination (Let's Encrypt), HTTP→HTTPS redirect |
| **Go Server** | API backend — fetches, parses, tests, caches, and serves proxy configs + WARP generation |
| **DuckDNS** | Keeps `belirofon-vpn.duckdns.org` pointed at your server IP |
| **Flutter Client** | Mobile app — fetches best config from API, establishes VPN connection, admin panel |
| **GeoIP DB** | MaxMind GeoLite2 database for country-level geo-filtering |

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Flutter 3.x (for mobile client)
- Docker & Docker Compose (for server deployment)
- A VPN subscription URL

### 1. Clone & Build

```bash
git clone https://github.com/belirofon/vpn.git
cd vpn

# Build the server
make build-server

# Or build + run locally with mock data (no subscription needed)
make run-server-mock
```

### 2. Test the Server

```bash
curl http://localhost:8080/health
# → {"status":"ready"}

curl http://localhost:8080/api/status
# → {"status":"ready","configs_tested":3,"best_name":"nl-1.example.com",...}
```

### 3. Run the Flutter Web Client

```bash
# With mock data server (already running):
make dev-mock
```

This starts the Flutter web app connected to `localhost:8080`. Connect/disconnect via the shield button. Long-press the shield to open **Debug Settings** and change the server URL.

## 📡 API Reference

All API endpoints are served by the Go backend on port `8080` (internal) or via Caddy reverse proxy on `443`/`8443`.

### `GET /health`

Server health check.

```json
{"status":"ready"}
```

Status values: `loading` · `testing` · `ready` · `error`

### `GET /api/status`

Server status with summary of tested configs.

```json
{
  "status": "ready",
  "message": "",
  "configs_tested": 42,
  "best_name": "nl-01.example.com",
  "best_latency_ms": 87,
  "updated": "2026-06-18T18:30:00Z"
}
```

Returns `503 Service Unavailable` when status is not `ready`.

### `GET /api/best-config`

Returns the **best performing** (lowest latency) **non-Russian** proxy configuration.

```json
{
  "config": {
    "id": "nl-01.example.com:443",
    "name": "🇳🇱 NL-01",
    "server": "203.0.113.10",
    "port": 443,
    "protocol": "vless",
    "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "tls": "tls",
    "network": "ws",
    "host": "example.com",
    "path": "/websocket",
    "latency_ms": 87,
    "country": "NL",
    "raw_link": "vless://...@nl-01.example.com:443?..."
  },
  "updated": "2026-06-18T18:30:00Z"
}
```

Returns `503` with `{"error":"no_available_configs","message":"..."}` if no configs available.

### `GET /api/configs`

Returns **all** tested and geo-filtered configurations (sorted by latency).

```json
{
  "configs": [ ... ],
  "total": 42,
  "updated": "2026-06-18T18:30:00Z"
}
```

Returns `503 Service Unavailable` when status is not `ready`.

### `POST /api/refresh`

Triggers an immediate refresh of the config cache (fetch → parse → test → geo → reality filter → sort).

```json
{"status": "refreshing"}
```

Returns `409 Conflict` if a refresh is already in progress.

### `GET /api/warp-config`

Returns the current Cloudflare WARP WireGuard config (if generated and WARP is enabled).

```json
{
  "config": {
    "protocol": "warp",
    "private_key": "...",
    "address_v4": "100.96.0.1/12",
    "address_v6": "2606:4700:110:.../128",
    "dns": "1.1.1.1",
    "server_public_key": "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
    "endpoint": "engage.cloudflareclient.com:2408",
    "client_id": "...",
    "latency_ms": 42
  },
  "updated": "2026-06-21T12:00:00Z"
}
```

Returns `404` with `{"error":"warp_not_available",...}` if WARP is disabled or not yet generated.

### Admin API

Authenticated endpoints under `/api/admin/`. Login first to obtain a bearer token:

```bash
curl -X POST https://your.domain/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"secret"}'
# → {"token":"..."}
```

All subsequent requests require `Authorization: Bearer <token>`.

#### `GET /api/admin/health`

Server health with uptime and config details.

```json
{
  "status": "ready",
  "message": "",
  "configs_tested": 42,
  "uptime": "12h34m56s",
  "subscription_url": "https://...",
  "refresh_interval": "30m0s"
}
```

#### `GET /api/admin/config`

Runtime configuration values.

```json
{
  "subscription_url": "https://...",
  "refresh_interval": "30m0s",
  "ping_timeout": "5s",
  "mock_configs": false,
  "skip_verify_tls": true,
  "cors_origins": "*"
}
```

#### `PUT /api/admin/config`

Update runtime config fields (both optional).

```json
{"subscription_url": "https://...", "refresh_interval": "30m"}
```

#### `POST /api/admin/refresh-configs`

Triggers an async config cache refresh.

```json
{"status": "refreshing"}
```

#### `GET /api/admin/endpoints`

Lists all registered API routes (methods + paths).

```json
{"endpoints": [...], "total": 25}
```

#### WARP Management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/warp` | Current WARP status (`{available, config}`) |
| `POST` | `/api/admin/warp/generate` | Force re-generate WARP config |
| `DELETE` | `/api/admin/warp` | Clear cached WARP config |

#### `POST /api/admin/logout`

Invalidates the current admin session token.

```json
{"status": "logged_out"}
```

## 📱 Mobile Client

The Flutter client is a cross-platform mobile app that connects to the VPN server and establishes a V2Ray-based VPN tunnel on your device.

### Screens (UI Preview)

| State | Screen |
|---|---|
| **Disconnected** | Shield icon (grey) + "DISCONNECT" button |
| **Connecting** | Pulse animation + spinner |
| **Connected** | Shield icon (green) + server info card + "DISCONNECT" button |

**Debug Menu**: Long-press the shield icon to open debug settings and change the server URL.

**Admin Panel**: Navigate to the admin screen to manage server configuration, monitor health, view API endpoints, control WARP config (generate/test/delete), and update runtime settings. Access requires admin credentials.

### Download

<div align="center">

| Platform | Download | Build Command |
|---|---|---|
| <img src="https://img.shields.io/badge/Android-3DDC84?style=for-the-badge&logo=android&logoColor=white" alt="Android" /> | [Download APK (latest)](https://github.com/belirofon/vpn/releases/latest/download/vpn-client-android.apk) | `make build-android-release` |
| <img src="https://img.shields.io/badge/iOS-000000?style=for-the-badge&logo=apple&logoColor=white" alt="iOS" /> | [Download IPA (latest)](https://github.com/belirofon/vpn/releases/latest/download/vpn-client-ios.ipa) | `make build-ios-release` |
| <img src="https://img.shields.io/badge/Web-4285F4?style=for-the-badge&logo=googlechrome&logoColor=white" alt="Web" /> | Runs via `make dev-mock` | `flutter run -d chrome` |

</div>

> **Note**: iOS builds require macOS with an active Apple Developer account.  
> Android release builds require a signing keystore (see [Android Signing](#android-signing) below).

### Building from Source

```bash
# Android debug APK (default server URL from build)
make build-android

# Android debug with custom server URL
make build-android SERVER_URL=https://belirofon-vpn.duckdns.org:8443

# Android release APK (requires signing config)
make build-android-release

# iOS release IPA (macOS only, requires developer account)
make build-ios-release
```

### Android Signing

For release builds, create `client/android/key.properties`:

```properties
storePassword=your_store_password
keyPassword=your_key_password
keyAlias=your_key_alias
storeFile=../keystore.jks
```

Then:
```bash
make build-android-release
```

The APK will be at `client/build/app/outputs/flutter-apk/app-release.apk`.

## 🚢 Deploy

### Using `make deploy` (rsync + Docker)

```bash
# Requirements:
#   - Remote server reachable via SSH on port 1337
#   - .env file on the remote server (~/vpn-server/.env)
#   - DuckDNS token in .env (DUCK_DNS_TOKEN)

make deploy
```

This will:
1. Rsync the `server/` directory to `~/vpn-server/` on the remote
2. Build the Docker image (if needed)
3. Start all services via Docker Compose

### Services

| Service | Port | Description |
|---|---|---|
| **Caddy** | `80` (HTTP → HTTPS redirect) | Reverse proxy + TLS |
| **Caddy** | `8443` (HTTPS) | Encrypted API access |
| **Go Server** | `8080` (internal, via Caddy) | API backend |

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|---|
| `SUBSCRIPTION_URL` | ✅ | — | URL to fetch proxy configs from |
| `DUCK_DNS_TOKEN` | ✅ | — | DuckDNS API token for auto DNS update |
| `DOMAIN` | — | `belirofon-vpn.duckdns.org` | Domain for HTTPS cert |
| `LISTEN_ADDR` | — | `:8080` | Internal server listen address |
| `REFRESH_INTERVAL` | — | `30m` | Config cache refresh interval |
| `PING_TIMEOUT` | — | `5s` | Ping timeout per config |
| `MOCK_CONFIGS` | — | `false` | Use mock configs (for testing) |
| `SKIP_VERIFY_TLS` | — | `true` | Skip TLS certificate verification (proxy testing compat) |
| `CORS_ORIGINS` | — | `*` | Allowed CORS origins |
| `ADMIN_EMAIL` | — | — | Admin login email (auth required for admin endpoints) |
| `ADMIN_PASSWORD` | — | — | Admin login password |
| `WARP_ENABLED` | — | `false` | Enable Cloudflare WARP WireGuard config generation |

### CI/CD (GitHub Actions)

| Workflow | Trigger | Action |
|---|---|---|
| **Deploy** | Push to `master` changing `server/**` | SCP → Docker build → restart → health check |
| **Build Android** | Push tag `v*` | Build APK → Create Release → Upload asset |

The latest Android APK is always available at:
```
https://github.com/belirofon/vpn/releases/latest/download/vpn-client-android.apk
```

## 🧪 Testing

```bash
# Integration tests (auto starts/stops server)
make test-integration

# Run with mock configs for local testing
make run-server-mock

# Check server health
make health

# View server logs (local)
make logs

# View remote logs
make deploy-logs
```

## 📁 Project Structure

```
vpn/
├── server/                          # Go backend
│   ├── cmd/server/main.go           # Entry point
│   ├── internal/
│   │   ├── cache/cache.go           # Config cache with periodic refresh
│   │   ├── config/config.go         # Environment config loader
│   │   ├── config/config_test.go    # Config tests (11)
│   │   ├── fetcher/fetcher.go       # HTTP subscription fetcher
│   │   ├── geo/geo.go               # GeoIP lookup & RU filtering
│   │   ├── geo/geo_test.go          # GeoIP tests (6)
│   │   ├── handler/handlers.go      # Public API handlers (Gin)
│   │   ├── handler/admin.go         # Admin API handlers (auth, config, WARP)
│   │   ├── model/models.go          # Data models & status types
│   │   ├── parser/parser.go         # VLESS/VMess/Trojan/SS parser
│   │   ├── parser/parser_test.go    # Parser tests (24)
│   │   ├── pipeline/pipeline.go     # Config processing pipeline
│   │   ├── pipeline/pipeline_test.go # Pipeline tests (6)
│   │   ├── resolver/resolver.go     # DNS resolver
│   │   ├── resolver/resolver_test.go # Resolver tests
│   │   ├── tester/                  # Connectivity tester
│   │   │   ├── tester.go            # TCP/TLS/WS testing
│   │   │   ├── tester_test.go       # Tester tests (6)
│   │   │   └── vless.go             # VLESS proxy test
│   │   └── warp/warp.go             # Cloudflare WARP config generation
│   ├── Caddyfile                    # Caddy reverse proxy config
│   ├── Dockerfile                   # Multi-stage Docker build
│   ├── docker-compose.yml           # Docker services
│   └── .env.example                 # Environment template
│
├── client/                          # Flutter mobile app
│   ├── lib/
│   │   ├── main.dart                # App entry point
│   │   ├── core/vpn/                # VPN service abstraction
│   │   │   ├── vpn_service.dart     # Abstract interface
│   │   │   ├── mobile_vpn_service.dart  # Sing-box-based mobile VPN
│   │   │   └── web_vpn_service.dart     # Web mock (UI testing)
│   │   ├── data/
│   │   │   ├── api/api_client.dart  # HTTP API client (Dio)
│   │   │   ├── dto/                 # Data transfer objects
│   │   │   │   ├── admin_models.dart    # Admin DTOs
│   │   │   │   └── vpn_config_dto.dart  # VPN config DTO
│   │   │   └── models/vpn_config.dart   # Config data model
│   │   ├── domain/entities/
│   │   │   ├── vpn_config.dart      # VPN config entity
│   │   │   └── warp_config.dart     # WARP config entity
│   │   └── presentation/
│   │       ├── screens/
│   │       │   ├── home_screen.dart     # Main UI (connect/disconnect)
│   │       │   └── admin_panel_screen.dart  # Admin panel (config, WARP)
│   │       └── viewmodels/
│   │           └── admin_viewmodel.dart  # Admin panel state/logic
│   ├── test/
│   │   ├── data/models/vpn_config_test.dart  # Model tests (12)
│   │   └── widget_test.dart                 # Widget tests
│   ├── android/                     # Android platform
│   └── ios/                         # iOS platform
│
├── .github/workflows/               # CI/CD pipelines
│   ├── deploy.yml                   # Deploy on push to master
│   ├── build-android.yml            # APK build on tag push
│   └── build-ios.yml                # iOS build (requires macOS runner)
├── Makefile                         # Build & deploy commands
├── PLAN.md                          # Architecture & implementation details
├── TECH_DEBT.md                     # Technical debt registry
├── TODO.md                          # Work plan
└── README.md                        # This file
```

## 🔧 Development

```bash
# One-command dev (build server + start Flutter web)
make dev SUBSCRIPTION_URL="your_subscription_url"

# Dev with mock configs (no subscription needed)
make dev-mock

# Build only the server
make build-server

# Run server with mock data
make run-server-mock

# Run server with real subscription
make run-server SUBSCRIPTION_URL="your_subscription_url"
```

## 🛣️ Roadmap

### ✅ Completed
- Multi-protocol parser (VLESS, VMess, Trojan, SS)
- Connectivity tester with real protocol handshakes (TCP/TLS/WS/VLESS/Trojan/REALITY)
- Config processing pipeline (fetch → parse → test → geo → reality → sort)
- GeoIP filtering with non-RU preference + RU fallback
- DuckDNS auto DNS update
- Caddy reverse proxy with auto-HTTPS (Let's Encrypt HTTP-01)
- Flutter mobile client (Android — working, iOS — requires Apple Developer)
- CI/CD deployment pipeline (Docker + GitHub Actions)
- GitHub Actions: automated Android APK builds on tag push
- REALITY filter (skip unsupported configs for Flutter client)
- Unit tests for Go (parser, config, geo, tester, pipeline, resolver) and Dart (vpn_config)
- Configurable TLS verification (SKIP_VERIFY_TLS) and CORS origins (CORS_ORIGINS)
- Graceful shutdown (SIGINT/SIGTERM)
- Cloudflare WARP WireGuard config generation with device registration and latency test
- Admin panel: server config management (subscription URL, refresh interval)
- Admin panel: WARP config management (generate, view, delete)
- Admin API with bearer token auth and session management

### ⬜ Upcoming
- REALITY support in Flutter client (uTLS/Xray)
- Push notifications for config updates
- Multi-user support (per-user config cache)
- WireGuard protocol support
- Dark theme
- Auto-connect on startup

## 📄 License

[MIT](LICENSE) © belirofon
