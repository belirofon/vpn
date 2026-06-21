import 'dart:async';
import '../../domain/entities/vpn_config.dart';
import '../../domain/services/vpn_service.dart';

class WebVpnService implements VpnService {
  final StreamController<VpnConnectionState> _stateController =
      StreamController<VpnConnectionState>.broadcast();
  VpnConnectionState _currentState = VpnConnectionState.disconnected;

  @override
  VpnConnectionState get currentState => _currentState;

  @override
  Stream<VpnConnectionState> get state => _stateController.stream;

  @override
  Future<void> connect(VpnConfig config) async {
    _setState(VpnConnectionState.connecting);
    // Simulate connection delay for UI testing
    await Future.delayed(const Duration(seconds: 2));
    _setState(VpnConnectionState.connected);
  }

  @override
  Future<void> disconnect() async {
    await Future.delayed(const Duration(milliseconds: 500));
    _setState(VpnConnectionState.disconnected);
  }

  @override
  void dispose() {
    _stateController.close();
  }

  void _setState(VpnConnectionState newState) {
    _currentState = newState;
    _stateController.add(newState);
  }
}
