# Config Testing Methodology

This document describes how VPN configs are tested server-side.
The approach is modelled after **v2rayN's RealPing** — we validate that a proxy
fully works (protocol handshake + traffic forwarding) without requiring a
running V2Ray/sing-box core instance.

## Test Types (by protocol)

| Protocol | Test method | What it validates |
|---|---|---|
| **VLESS** | `testVlessProxy` | VLESS handshake (version + UUID + TCP command) → domain-based target (`www.gstatic.com:80`) → HTTP GET `/generate_204` → HTTP response |
| **Trojan** | `testTrojanProxy` | TLS handshake (if `security=tls`) → `[password]\r\n` → HTTP GET through tunnel → HTTP response |
| **VMESS / SS** | `testWsUpgrade` | TCP connect + TLS handshake (if `security=tls`) + WebSocket upgrade (if `type=ws`). No protocol-level test — covers only transport |
| **REALITY** | — | Skipped. REALITY requires uTLS (Go 1.24+) for proper fingerprinting |

## VLESS (deepest test)

The VLESS test follows v2rayN's principle of testing the **full proxy pipeline**:

```
TCP → (TLS) → VLESS handshake → proxy resolves domain → forwards HTTP → response
```

1. **VLESS request** uses address type `0x03` (domain), targeting `www.gstatic.com:80`
   — this tests DNS resolution through the proxy, matching v2rayN's `generate_204` approach
2. **HTTP GET** `/generate_204` is sent through the established tunnel
3. **Response** is checked for `HTTP/` prefix — any valid HTTP response (200, 204, 301, etc.)
   proves traffic is forwarded correctly

**Why domain-based target?** v2rayN sends HTTP requests to domains like
`www.google.com/generate_204`. Using a domain target (0x03) in the VLESS
request tests the proxy's DNS resolution, which IP-based (0x01) targets skip.

## Trojan

Simple password-authenticated tunnel:

```
TCP → (TLS) → [password]\r\n → HTTP GET /generate_204 → HTTP response
```

## VMESS / SS

Protocol-level testing is not implemented for VMESS and Shadowsocks due to
their encrypted payload formats. For these protocols, only transport is
validated:

- TCP connect to `server:port`
- TLS handshake (if `security=tls`)
- WebSocket upgrade (if `type=ws` or `network=ws`)

This is equivalent to v2rayN's **TCPing** for these protocols.

## Comparison with v2rayN

| Aspect | v2rayN RealPing | Our server test |
|---|---|---|
| **Core** | Full V2Ray/sing-box process | Protocol-level (no core needed) |
| **Target** | `https://google.com/generate_204` via SOCKS5 | `www.gstatic.com:80` via protocol handshake |
| **M probes** | 2 requests, takes min time | 1 request per config (parallel over 500 configs) |
| **Latency** | End-to-end through core | TCP + protocol handshake + HTTP response |
| **IP info** | Fetches proxy IP via API | — |

## Geolocation detection

Server location is determined **after** a successful proxy test using the
**resolved server IP** via a local MaxMind GeoLite2 Country database.

### Why not through the proxy (like v2rayN)?

v2rayN sends an HTTP request through the SOCKS5 proxy to `api.ip.sb/geoip`
to learn the **egress IP and country**. This requires a running V2Ray core.

Our server tests **2000+ configs** in parallel. Sending an additional HTTP
request through each proxy would:
- Add ~1-3s per config (DNS + HTTP through proxy)
- Hit rate limits on free geoip APIs (ip-api.com: 45/min)
- Require a running core per config (not feasible server-side)

### Current approach

```
resolveIP(server) → GeoIP lookup → country
```

This gives the **server's location**, not the egress IP. For subscription
configs (VLESS/Trojan/VMESS/SS), the server IS the proxy — the location
matches in the vast majority of cases.

### Scaling

2000 GeoIP lookups in an in-memory mmdb database = **<2ms total**.
No rate limits, no external dependencies.

### Fallback

If `GeoLite2-Country.mmdb` is not available, all configs pass through
without country filtering (`country = ""`). The database is auto-downloaded
on `make build-server` from the P3TERX community mirror.

## Test target

```
Domain: www.gstatic.com
Port:   80
Path:   /generate_204
```

`www.gstatic.com/generate_204` returns HTTP 204 No Content reliably.
It is one of the standard URLs v2rayN uses for ping testing.
