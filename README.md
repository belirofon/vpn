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
| **Go Server** | API backend — fetches, parses, tests, caches, and serves proxy configs |
| **DuckDNS** | Keeps `belirofon-vpn.duckdns.org` pointed at your server IP |
| **Flutter Client** | Mobile app — fetches best config from API and establishes VPN connection |
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

### `GET /api/configs`

Returns **all** tested and geo-filtered configurations (sorted by latency).

```json
{
  "configs": [ ... ],
  "total": 42,
  "updated": "2026-06-18T18:30:00Z"
}
```

### `POST /api/refresh`

Triggers an immediate refresh of the config cache (fetch → parse → test → geo-filter).

```json
{"status": "refreshing"}
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
|---|---|---|---|
| `SUBSCRIPTION_URL` | ✅ | — | URL to fetch proxy configs from |
| `DUCK_DNS_TOKEN` | ✅ | — | DuckDNS API token for auto DNS update |
| `DOMAIN` | — | `belirofon-vpn.duckdns.org` | Domain for HTTPS cert |
| `LISTEN_ADDR` | — | `:8080` | Internal server listen address |
| `REFRESH_INTERVAL` | — | `30m` | Config cache refresh interval |
| `PING_TIMEOUT` | — | `5s` | Ping timeout per config |
| `MOCK_CONFIGS` | — | `false` | Use mock configs (for testing) |

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
│   │   ├── fetcher/fetcher.go       # HTTP subscription fetcher
│   │   ├── geo/geo.go               # GeoIP lookup & RU filtering
│   │   ├── handler/handlers.go      # HTTP API handlers (Gin)
│   │   ├── model/models.go          # Data models & status types
│   │   ├── parser/parser.go         # VLESS/VMess/Trojan/SS parser
│   │   ├── resolver/resolver.go     # DNS resolver
│   │   └── tester/                  # Connectivity tester
│   │       ├── tester.go            # TCP/TLS/WS testing
│   │       └── vless.go             # VLESS proxy test
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
│   │   │   ├── mobile_vpn_service.dart  # V2Ray-based mobile VPN
│   │   │   └── web_vpn_service.dart     # Web mock (UI testing)
│   │   ├── data/
│   │   │   ├── api/api_client.dart  # HTTP API client (Dio)
│   │   │   └── models/vpn_config.dart   # Config data model
│   │   └── presentation/screens/
│   │       └── home_screen.dart     # Main UI (connect/disconnect)
│   ├── android/                     # Android platform
│   └── ios/                         # iOS platform
│
├── .github/workflows/deploy.yml     # CI/CD pipeline
├── Makefile                         # Build & deploy commands
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

- [x] Multi-protocol parser (VLESS, VMess, Trojan, SS)
- [x] Connectivity tester with real protocol handshakes
- [x] GeoIP filtering with non-RU preference + RU fallback
- [x] DuckDNS auto DNS update
- [x] Caddy reverse proxy with auto-HTTPS (Let's Encrypt HTTP-01)
- [x] Flutter mobile client (Android — working, iOS — requires Apple Developer)
- [x] CI/CD deployment pipeline (Docker + GitHub Actions)
- [x] GitHub Actions: automated Android APK builds on tag push
- [x] REALITY filter (skip unsupported configs for Flutter client)
- [ ] REALITY support in Flutter client (uTLS/Xray)
- [ ] Push notifications for config updates
- [ ] Multi-user support (per-user config cache)
- [ ] WireGuard protocol support

## 📄 License

[MIT](LICENSE) © belirofon
