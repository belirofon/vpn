import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter_singbox_client/flutter_singbox_client.dart';
import '../../domain/entities/vpn_config.dart';
import '../../domain/entities/warp_config.dart';
import '../../domain/services/vpn_service.dart';

class MobileVpnService implements VpnService {
  final SingboxClient _client = SingboxClient();
  final StreamController<VpnConnectionState> _stateController =
      StreamController<VpnConnectionState>.broadcast();
  VpnConnectionState _currentState = VpnConnectionState.disconnected;
  bool _initialized = false;
  StreamSubscription<ServiceState>? _stateSub;
  StreamSubscription<String>? _faultSub;

  @override
  VpnConnectionState get currentState => _currentState;

  @override
  Stream<VpnConnectionState> get state => _stateController.stream;

  Future<void> initialize() async {
    if (_initialized) {
      debugPrint('[MobileVpnService] already initialized, skip');
      return;
    }

    debugPrint('[MobileVpnService] initialize() — calling SingboxClient.initialize()');
    await _client.initialize();
    debugPrint('[MobileVpnService] SingboxClient.initialize() OK');

    _stateSub = _client.serviceStateStream.listen(_onServiceStateChanged);
    _faultSub = _client.faultStream.listen(_onFaultChanged);
    debugPrint('[MobileVpnService] state & fault streams subscribed');

    _initialized = true;
  }

  @override
  Future<void> connect(VpnConfig config) async {
    debugPrint('[MobileVpnService] connect() — config=${config.id}');

    final singboxJson = _buildFullConfig(config);
    if (singboxJson == null) {
      debugPrint('[MobileVpnService] connect() — FAIL: no sing-box config available');
      _setState(VpnConnectionState.error);
      throw Exception('No Sing-box config available');
    }

    _setState(VpnConnectionState.connecting);

    try {
      if (!_initialized) {
        debugPrint('[MobileVpnService] connect() — calling initialize()');
        await initialize();
      }

      debugPrint('[MobileVpnService] connect() — requesting VPN permission');
      final bool allowed = await _client.requestVPNPermission();
      debugPrint('[MobileVpnService] connect() — VPN permission allowed=$allowed');
      if (!allowed) {
        _setState(VpnConnectionState.error);
        throw Exception('VPN permission denied');
      }

      // Validate config
      debugPrint('[MobileVpnService] connect() — calling checkConfig()');
      try {
        await _client.checkConfig(singboxJson);
        debugPrint('[MobileVpnService] connect() — checkConfig() OK');
      } catch (e) {
        debugPrint('[MobileVpnService] connect() — checkConfig() FAILED: $e');
        _setState(VpnConnectionState.error);
        rethrow;
      }

      // Create session options
      final sessionOptions = SessionOptions(
        config: singboxJson,
        notification: NotificationConfig(
          title: config.name,
          showTrafficStats: true,
          showStopButton: true,
          stopButtonLabel: 'Disconnect',
        ),
      );

      debugPrint('[MobileVpnService] connect() — calling SingboxClient.connect()');
      await _client.connect(sessionOptions);
      debugPrint('[MobileVpnService] connect() — connect() returned, waiting for state stream');
    } catch (e) {
      debugPrint('[MobileVpnService] connect() — EXCEPTION: $e');
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  /// Builds a full Sing-box JSON config wrapping the server-provided outbound.
  /// Falls back to parsing raw_link if singboxConfig is not available.
  String? _buildFullConfig(VpnConfig config) {
    final outbound = config.singboxConfig;

    if (outbound != null) {
      debugPrint('[MobileVpnService] _buildFullConfig — using server-provided singboxConfig');
      return jsonEncode({
        'inbounds': [
          {
            'type': 'tun',
            'tag': 'tun-in',
            'address': [
              '172.19.0.1/30',
            ],
            'auto_route': true,
            'strict_route': true,
          },
        ],
        'outbounds': [outbound],
        'route': {
          'auto_detect_interface': true,
        },
      });
    }

    // Fallback: build an outbound from raw_link (legacy).
    debugPrint('[MobileVpnService] _buildFullConfig — WARN: singboxConfig missing, parsing raw_link');
    final rawLink = config.rawLink;
    if (rawLink == null || rawLink.isEmpty) return null;

    try {
      final uri = Uri.parse(rawLink);
      final query = uri.queryParameters;
      final fallbackOutbound = <String, dynamic>{
        'type': uri.scheme,
        'tag': 'proxy',
        'server': uri.host,
        'server_port': uri.port,
      };

      if (uri.scheme == 'vless') {
        fallbackOutbound['uuid'] = uri.userInfo;
        fallbackOutbound['flow'] = '';
        if (query['security'] == 'tls') {
          fallbackOutbound['tls'] = {
            'enabled': true,
            'server_name': query['sni'] ?? query['host'] ?? uri.host,
            'insecure': true,
          };
        }
      } else if (uri.scheme == 'trojan') {
        fallbackOutbound['password'] = uri.userInfo;
      } else if (uri.scheme == 'vmess') {
        fallbackOutbound['uuid'] = uri.userInfo;
        fallbackOutbound['security'] = 'auto';
      } else if (uri.scheme == 'ss') {
        fallbackOutbound['password'] = uri.userInfo;
        fallbackOutbound['method'] = 'aes-256-gcm';
      }

      if (query['type'] != null && query['type'] != 'tcp') {
        final transport = <String, dynamic>{'type': query['type']};
        if (query['path'] != null) transport['path'] = query['path'];
        if (query['host'] != null) {
          transport['headers'] = {'Host': query['host']};
        }
        fallbackOutbound['transport'] = transport;
      }

      return jsonEncode({
        'inbounds': [
          {
            'type': 'tun',
            'tag': 'tun-in',
            'address': [
              '172.19.0.1/30',
            ],
            'auto_route': true,
            'strict_route': true,
          },
        ],
        'outbounds': [fallbackOutbound],
        'route': {
          'auto_detect_interface': true,
        },
      });
    } catch (e) {
      debugPrint('[MobileVpnService] _buildFullConfig — fallback failed: $e');
      return null;
    }
  }

  @override
  Future<void> connectWarp(WarpConfig config) async {
    debugPrint('[MobileVpnService] connectWarp() — endpoint=${config.endpoint}');

    _setState(VpnConnectionState.connecting);

    try {
      if (!_initialized) {
        debugPrint('[MobileVpnService] connectWarp() — calling initialize()');
        await initialize();
      }

      debugPrint('[MobileVpnService] connectWarp() — requesting VPN permission');
      final bool allowed = await _client.requestVPNPermission();
      debugPrint('[MobileVpnService] connectWarp() — VPN permission allowed=$allowed');
      if (!allowed) {
        _setState(VpnConnectionState.error);
        throw Exception('VPN permission denied');
      }

      // Build sing-box WireGuard endpoint JSON from WarpConfig.
      // sing-box 1.14 (embedded in flutter_singbox_client) requires WireGuard
      // as an endpoint (not an outbound). Migration from deprecated outbound:
      //   outbounds[] → endpoints[]
      //   local_address → address
      //   server/server_port → peers[].address/peers[].port
      //   peer_public_key → peers[].public_key
      //   reserved → peers[].reserved
      //
      // address requires CIDR notation (/32 for v4, /128 for v6).
      // Ensure it even if server returns a bare IP.
      String cidr(String addr) {
        if (addr.contains('/')) return addr;
        return addr.contains(':') ? '$addr/128' : '$addr/32';
      }

      final parts = config.endpoint.split(':');
      final host = parts[0];
      final port = parts.length > 1 ? int.parse(parts[1]) : 2408;

      final warpJson = jsonEncode({
        'inbounds': [
          {
            'type': 'tun',
            'tag': 'tun-in',
            'address': ['172.19.0.1/30'],
            'auto_route': true,
            'strict_route': true,
          }
        ],
        'outbounds': [
          {
            'type': 'selector',
            'tag': 'warp-out',
            'outbounds': ['warp-ep'],
          },
        ],
        'endpoints': [
          {
            'type': 'wireguard',
            'tag': 'warp-ep',
            'address': [cidr(config.addressV4), cidr(config.addressV6)],
            'private_key': config.privateKey,
            'peers': [
              {
                'address': host,
                'port': port,
                'public_key': config.serverPublicKey,
                'allowed_ips': ['0.0.0.0/0', '::/0'],
                'reserved': [0, 0, 0],
              },
            ],
            'mtu': 1280,
          },
        ],
        'route': {
          'auto_detect_interface': true,
        },
      });

      debugPrint('[MobileVpnService] connectWarp() — JSON config built, length=${warpJson.length}');

      final sessionOptions = SessionOptions(
        config: warpJson,
        notification: const NotificationConfig(
          title: 'WARP',
          showTrafficStats: true,
          showStopButton: true,
          stopButtonLabel: 'Disconnect',
        ),
      );

      debugPrint('[MobileVpnService] connectWarp() — calling SingboxClient.connect()');
      await _client.connect(sessionOptions);
      debugPrint('[MobileVpnService] connectWarp() — connect() returned, waiting for state stream');
    } catch (e) {
      debugPrint('[MobileVpnService] connectWarp() — EXCEPTION: $e');
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  @override
  Future<void> disconnect() async {
    debugPrint('[MobileVpnService] disconnect()');
    try {
      await _client.disconnect();
    } finally {
      _setState(VpnConnectionState.disconnected);
    }
  }

  @override
  void dispose() {
    debugPrint('[MobileVpnService] dispose()');
    _stateSub?.cancel();
    _faultSub?.cancel();
    _stateController.close();
  }

  void _onServiceStateChanged(ServiceState state) {
    debugPrint('[MobileVpnService] _onServiceStateChanged: $state');
    switch (state) {
      case ServiceState.started:
        _setState(VpnConnectionState.connected);
        return;
      case ServiceState.starting:
        _setState(VpnConnectionState.connecting);
        return;
      case ServiceState.stopped:
        _setState(VpnConnectionState.disconnected);
        return;
      case ServiceState.stopping:
        _setState(VpnConnectionState.disconnected);
        return;
    }
  }

  void _onFaultChanged(String error) {
    debugPrint('[MobileVpnService] _onFaultChanged: $error');
    _setState(VpnConnectionState.error);
  }

  void _setState(VpnConnectionState newState) {
    debugPrint('[MobileVpnService] _setState: $_currentState -> $newState');
    if (_currentState == newState) return;
    _currentState = newState;
    if (!_stateController.isClosed) {
      _stateController.add(newState);
    }
  }
}
