import 'dart:async';
import 'package:flutter_v2ray_plus/flutter_v2ray.dart';
import '../../domain/entities/vpn_config.dart';
import '../../domain/services/vpn_service.dart';

class MobileVpnService implements VpnService {
  final FlutterV2ray _v2ray = FlutterV2ray();
  final StreamController<VpnConnectionState> _stateController =
      StreamController<VpnConnectionState>.broadcast();
  VpnConnectionState _currentState = VpnConnectionState.disconnected;
  bool _initialized = false;
  StreamSubscription<VlessStatus>? _statusSub;

  @override
  VpnConnectionState get currentState => _currentState;

  @override
  Stream<VpnConnectionState> get state => _stateController.stream;

  Future<void> initialize({
    String providerBundleIdentifier = 'com.example.vpn.VPNProvider',
    String groupIdentifier = 'group.com.example.vpn',
  }) async {
    if (_initialized) return;

    await _v2ray.initializeVless(
      providerBundleIdentifier: providerBundleIdentifier,
      groupIdentifier: groupIdentifier,
    );

    _statusSub = _v2ray.onStatusChanged.listen(_onStatusChanged);

    _initialized = true;
  }

  @override
  Future<void> connect(VpnConfig config) async {
    if (config.rawLink == null || config.rawLink!.isEmpty) {
      _setState(VpnConnectionState.error);
      throw Exception('No raw config link provided');
    }

    _setState(VpnConnectionState.connecting);

    try {
      if (!_initialized) {
        await initialize();
      }

      bool allowed = await _v2ray.requestPermission();
      if (!allowed) {
        _setState(VpnConnectionState.error);
        throw Exception('VPN permission denied');
      }

      final parsed = FlutterV2ray.parseFromURL(config.rawLink!);

      // Fix 1: Enable DNS sniffing so V2Ray intercepts DNS queries
      // Without this, UDP DNS requests hit the TCP-only VLESS outbound and get dropped
      parsed.inbound["sniffing"] = {
        "enabled": true,
        "destOverride": ["http", "tls"],
      };

      // Fix 2: Set DNS servers (V2Ray uses these internally for sniffing)
      parsed.dns = {
        "servers": [
          "https://1.1.1.1/dns-query", // DNS-over-HTTPS bypasses carrier blocking
          "1.1.1.1",
        ],
      };

      // Fix 3: Explicit routing — UDP bypasses proxy (VLESS+WS is TCP-only),
      // TCP goes through proxy
      parsed.routing = {
        "domainStrategy": "UseIp",
        "rules": [
          {
            "type": "field",
            "network": "udp",
            "outboundTag": "direct",
          },
          {
            "type": "field",
            "network": "tcp",
            "outboundTag": "proxy",
          },
        ],
      };

      final configJson = parsed.getFullConfiguration();

      await _v2ray.startVless(
        remark: config.name,
        config: configJson,
        // Also set DNS on the native VPN tunnel to match
        dnsServers: ["1.1.1.1"],
      );
    } catch (e) {
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  @override
  Future<void> disconnect() async {
    try {
      await _v2ray.stopVless();
    } finally {
      _setState(VpnConnectionState.disconnected);
    }
  }

  @override
  void dispose() {
    _statusSub?.cancel();
    _stateController.close();
  }

  void _onStatusChanged(VlessStatus status) {
    switch (status.state.toUpperCase()) {
      case 'CONNECTED':
        _setState(VpnConnectionState.connected);
        return;
      case 'CONNECTING':
        _setState(VpnConnectionState.connecting);
        return;
      case 'DISCONNECTED':
        _setState(VpnConnectionState.disconnected);
        return;
      default:
        if (status.state.isNotEmpty) {
          _setState(VpnConnectionState.error);
        }
        return;
    }
  }

  void _setState(VpnConnectionState newState) {
    if (_currentState == newState) return;
    _currentState = newState;
    if (!_stateController.isClosed) {
      _stateController.add(newState);
    }
  }
}
