import 'dart:async';
import '../entities/vpn_config.dart';
import '../entities/warp_config.dart';

enum VpnConnectionState {
  disconnected,
  connecting,
  connected,
  error,
}

abstract class VpnService {
  Future<void> connect(VpnConfig config);
  Future<void> connectWarp(WarpConfig config);
  Future<void> disconnect();
  Stream<VpnConnectionState> get state;
  VpnConnectionState get currentState;
  void dispose();
}
