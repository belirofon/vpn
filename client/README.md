# vpn_client

A VPN client application using `flutter_singbox_client` for secure connectivity.

## Getting Started

This project is a starting point for a Flutter application.

A few resources to get you started if this is your first Flutter project:

- [Learn Flutter](https://docs.flutter.dev/get-started/learn-flutter)
- [Write your first Flutter app](https://docs.flutter.dev/get-started/codelab)
- [Flutter learning resources](https://docs.flutter.dev/reference/learning-resources)

For help getting started with Flutter development, view the
[online documentation](https://docs.flutter.dev/), which offers tutorials,
samples, guidance on mobile development, and a full API reference.

## Project Overview

This VPN client uses `flutter_singbox_client` to establish secure VPN connections. The application provides:

- **One-tap VPN connection** - Simple connect/disconnect interface
- **Server selection** - Automatically selects the best available server
- **Real-time status** - Shows connection status and server details
- **Debug mode** - Allows manual server URL configuration
- **Admin panel** - Server configuration and monitoring

## Key Features

- **VPN Mode** - Full device-wide TUN tunnel with kill switch
- **Proxy Mode** - HTTP/SOCKS proxy without VPN permission
- **Clash API** - Runtime mode switching and outbound group control
- **Per-App Proxy** - Include/exclude specific apps from the tunnel
- **System Proxy** - Register HTTP inbound as device-wide proxy on Android Q+
- **Kill Switch** - Block all traffic at the OS level when the tunnel is down
- **Live Traffic** - Real-time upload/download speeds and session totals
- **Connection Tracking** - Full per-connection metadata, lifecycle, and close control
- **Live Logs** - Real-time Go core log streaming with per-level filtering
- **Fault Alerts** - Dedicated fault stream for actionable service errors
- **Memory Limits** - Go runtime soft memory cap with optional connection kill
- **Network Testing** - Built-in STUN (NAT type, latency) and network quality tests
- **Config Validation** - Go core validation and JSON formatting before connecting
- **Hot Reload** - Reload config without restarting the service or dropping connections
- **Boot Auto-start** - Boot broadcast relay so your app can reconnect after device restart

## Requirements

- Flutter SDK `>=3.19.0`
- Dart SDK `>=3.3.0`
- Android `minSdk` 23 (Marshmallow)
- Android `compileSdk` 35
- Java target 17

## Installation

**1. Add the dependency** (`pubspec.yaml`):

```yaml
dependencies:
  flutter:
    sdk: flutter
  flutter_singbox_client: ^1.0.1
```

**2. Configure Android Gradle** (`android/app/build.gradle`):

```gradle
android {
    compileSdk 35
    defaultConfig { minSdk 23 }
    compileOptions {
        sourceCompatibility JavaVersion.VERSION_17
        targetCompatibility JavaVersion.VERSION_17
    }
    kotlinOptions { jvmTarget = "17" }
}
```

## Quick Start

```dart
import 'package:flutter_singbox_client/flutter_singbox_client.dart';

final client = SingboxClient();

// Initialize once at app startup
await client.initialize();

// Request VPN permission (VPN mode only)
if (!await client.requestVPNPermission()) return;

// Validate config — throws with the Go core's error message on failure
try {
  await client.checkConfig(myConfigJson);
} catch (e) {
  showError('$e');
  return;
}

// Connect
await client.connect(SessionOptions(
  config: myConfigJson,
  networkMode: NetworkMode.vpn,
  notification: NotificationConfig(
    title: 'My VPN',
    showTrafficStats: true,
    showStopButton: true,
    stopButtonLabel: 'Disconnect',
  ),
));

// Subscribe to live events
client.serviceStateStream.listen((state) => print('State: $state'));
client.trafficStatsStream.listen((s) => print('↑${s.uplinkBps} ↓${s.downlinkBps}'));
client.faultStream.listen((error) => showSnackbar(error));

// Disconnect
await client.disconnect();
```

## VPN Mode vs Proxy Mode

Select the operating mode via `SessionOptions.networkMode`.

VPN Mode

Proxy Mode

**Android service**

`VpnService`

Foreground `Service`

**TUN device**

✅

❌

**Traffic capture**

System-wide at OS level

Manual (apps must use the proxy)

**VPN permission required**

✅

❌

**Kill switch**

✅

❌

**System proxy**

✅ (Android Q+)

❌

**Per-app routing**

✅

❌

**`tun` inbound in config**

Required for auto-route

Must not be present

Use `NetworkMode.vpn` when your config uses `"type": "tun"` or `"auto_route": true`, or when you need kill switch, per-app routing, or system proxy.

Use `NetworkMode.proxy` when you only need HTTP/SOCKS proxy ports without requesting VPN permission.

Warning

Proxy mode configs must **not** contain a `tun` inbound. Starting proxy mode with a TUN config causes an immediate startup failure via `faultStream`.

## Documentation

Guide

Contents

[Getting Started](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/getting-started.md)

Initialization, lifecycle, config validation, traffic, outbound groups, Clash mode, connections, network testing, system proxy, boot

[API Reference](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/api-reference.md)

All methods and event streams

[Data Models](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/models.md)

Field reference for every SDK type

[Best Practices](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/best-practices.md)

Integration patterns, pitfalls, and a full example

[Supported Protocols & Features](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/supported-protocols.md)

All Sing-box protocols, transports, TLS, DNS, routing, and obfuscation

[Android Permissions](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/android-permissions.md)

Required, optional, and sensitive permissions

[Troubleshooting](https://github.com/amir-zr/flutter_singbox_client/blob/main/doc/troubleshooting.md)

Common errors and fixes

## License

This project is licensed under the [GNU General Public License v3.0](https://github.com/amir-zr/flutter_singbox_client/blob/main/LICENSE) (GPL-3.0).
