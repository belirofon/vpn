import 'dart:async';
import 'package:flutter_v2ray_client/flutter_v2ray.dart';
import '../../data/models/vpn_config.dart';
import 'vpn_service.dart';

class MobileVpnService implements VpnService {
  late final V2ray _v2ray;
  final StreamController<VpnConnectionState> _stateController =
      StreamController<VpnConnectionState>.broadcast();
  VpnConnectionState _currentState = VpnConnectionState.disconnected;
  bool _initialized = false;

  @override
  VpnConnectionState get currentState => _currentState;

  @override
  Stream<VpnConnectionState> get state => _stateController.stream;

  MobileVpnService() {
    _v2ray = V2ray(onStatusChanged: _onStatusChanged);
  }

  Future<void> initialize() async {
    if (_initialized) return;
    await _v2ray.initialize();
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

      final parsed = V2ray.parseFromURL(config.rawLink!);

      // Enable DNS sniffing so V2Ray intercepts DNS queries
      parsed.inbound["sniffing"] = {
        "enabled": true,
        "destOverride": ["http", "tls"],
      };

      // Set DNS servers
      parsed.dns = {
        "servers": [
          "https://1.1.1.1/dns-query",
          "1.1.1.1",
        ],
      };

      // For TCP-only protocols (WS, etc.) — route UDP directly, TCP through proxy.
      // For REALITY (XTLS Vision) — skip override, XTLS handles both TCP/UDP.
      if (config.tls != 'reality') {
        parsed.routing = {
          "domainStrategy": "UseIp",
          "rules": [
            {"type": "field", "network": "udp", "outboundTag": "direct"},
            {"type": "field", "network": "tcp", "outboundTag": "proxy"},
          ],
        };
      }

      final configJson = parsed.getFullConfiguration();

      await _v2ray.startV2Ray(
        remark: config.name,
        config: configJson,
      );
    } catch (e) {
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  @override
  Future<void> disconnect() async {
    try {
      await _v2ray.stopV2Ray();
    } finally {
      _setState(VpnConnectionState.disconnected);
    }
  }

  @override
  void dispose() {
    _stateController.close();
  }

  void _onStatusChanged(V2RayStatus status) {
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
