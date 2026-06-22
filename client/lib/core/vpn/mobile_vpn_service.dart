import 'dart:async';
import 'dart:convert';
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
    if (_initialized) return;

    await _client.initialize();

    _stateSub = _client.serviceStateStream.listen(_onServiceStateChanged);
    _faultSub = _client.faultStream.listen(_onFaultChanged);

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

      final bool allowed = await _client.requestVPNPermission();
      if (!allowed) {
        _setState(VpnConnectionState.error);
        throw Exception('VPN permission denied');
      }

      // Validate config
      try {
        await _client.checkConfig(config.rawLink!);
      } catch (e) {
        _setState(VpnConnectionState.error);
        rethrow;
      }

      // Create session options
      final sessionOptions = SessionOptions(
        config: config.rawLink!,
        notification: NotificationConfig(
          title: config.name,
          showTrafficStats: true,
          showStopButton: true,
          stopButtonLabel: 'Disconnect',
        ),
      );

      await _client.connect(sessionOptions);
    } catch (e) {
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  @override
  Future<void> connectWarp(WarpConfig config) async {
    _setState(VpnConnectionState.connecting);

    try {
      if (!_initialized) {
        await initialize();
      }

      final bool allowed = await _client.requestVPNPermission();
      if (!allowed) {
        _setState(VpnConnectionState.error);
        throw Exception('VPN permission denied');
      }

      // Build sing-box WireGuard JSON from WarpConfig
      final parts = config.endpoint.split(':');
      final host = parts[0];
      final port = parts.length > 1 ? int.parse(parts[1]) : 2408;

      final warpJson = jsonEncode({
        'inbounds': [
          {
            'type': 'tun',
            'tag': 'tun-in',
            'inet4_address': '172.19.0.1/30',
            'auto_route': true,
            'strict_route': true,
          },
        ],
        'outbounds': [
          {
            'type': 'wireguard',
            'tag': 'warp',
            'server': host,
            'server_port': port,
            'local_address': [
              config.addressV4,
              config.addressV6,
            ],
            'private_key': config.privateKey,
            'peer_public_key': config.serverPublicKey,
            'mtu': 1280,
          },
        ],
        'route': {
          'auto_detect_interface': true,
        },
      });

      final sessionOptions = SessionOptions(
        config: warpJson,
        notification: const NotificationConfig(
          title: 'WARP',
          showTrafficStats: true,
          showStopButton: true,
          stopButtonLabel: 'Disconnect',
        ),
      );

      await _client.connect(sessionOptions);
    } catch (e) {
      _setState(VpnConnectionState.error);
      rethrow;
    }
  }

  @override
  Future<void> disconnect() async {
    try {
      await _client.disconnect();
    } finally {
      _setState(VpnConnectionState.disconnected);
    }
  }

  @override
  void dispose() {
    _stateSub?.cancel();
    _faultSub?.cancel();
    _stateController.close();
  }

  void _onServiceStateChanged(ServiceState state) {
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
    _setState(VpnConnectionState.error);
  }

  void _setState(VpnConnectionState newState) {
    if (_currentState == newState) return;
    _currentState = newState;
    if (!_stateController.isClosed) {
      _stateController.add(newState);
    }
  }
}
