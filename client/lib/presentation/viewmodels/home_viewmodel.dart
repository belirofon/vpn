import 'dart:async';
import 'package:flutter/foundation.dart';
import '../../data/api/api_client.dart';
import '../../domain/entities/vpn_config.dart';
import '../../domain/services/vpn_service.dart';

enum HomeMode { warp, proxy }

enum ScreenState { disconnected, connecting, connected, error }

class HomeViewModel extends ChangeNotifier {
  final ApiClient _apiClient;
  final VpnService _vpnService;
  StreamSubscription<VpnConnectionState>? _stateSub;

  ScreenState _screenState = ScreenState.disconnected;
  VpnConfig? _activeConfig;
  String? _errorMessage;
  HomeMode _selectedMode = HomeMode.proxy;

  HomeViewModel({
    required ApiClient apiClient,
    required VpnService vpnService,
  })  : _apiClient = apiClient,
        _vpnService = vpnService {
    _stateSub = _vpnService.state.listen(_onVpnStateChanged);
    _screenState = _vpnStateToScreen(_vpnService.currentState);
  }

  ScreenState _vpnStateToScreen(VpnConnectionState state) {
    return switch (state) {
      VpnConnectionState.disconnected => ScreenState.disconnected,
      VpnConnectionState.connecting => ScreenState.connecting,
      VpnConnectionState.connected => ScreenState.connected,
      VpnConnectionState.error => ScreenState.error,
    };
  }

  ScreenState get screenState => _screenState;
  VpnConfig? get activeConfig => _activeConfig;
  String? get errorMessage => _errorMessage;
  HomeMode get selectedMode => _selectedMode;

  bool get isConnected => _screenState == ScreenState.connected;
  bool get isLoading => _screenState == ScreenState.connecting;

  void selectMode(HomeMode mode) {
    if (_selectedMode == mode) return;
    _selectedMode = mode;
    notifyListeners();
  }

  Future<void> toggleConnection() async {
    if (_screenState == ScreenState.connected) {
      await _vpnService.disconnect();
      return;
    }

    _errorMessage = null;
    notifyListeners();

    try {
      final config = await _apiClient.getBestConfig();
      if (config == null) {
        _errorMessage =
            'Server unavailable.\nCheck that the server is running.';
        notifyListeners();
        return;
      }

      await _vpnService.connect(config);
      _activeConfig = config;
    } catch (e) {
      _errorMessage = e.toString();
      notifyListeners();
    }
  }

  String get statusLabel {
    return switch (_screenState) {
      ScreenState.connected => 'Connected',
      ScreenState.connecting => 'Connecting…',
      ScreenState.error => 'Error',
      ScreenState.disconnected => 'Disconnected',
    };
  }

  void _onVpnStateChanged(VpnConnectionState state) {
    _screenState = _vpnStateToScreen(state);
    if (state == VpnConnectionState.disconnected) {
      _activeConfig = null;
      _errorMessage = null;
    }
    notifyListeners();
  }

  @override
  void dispose() {
    _stateSub?.cancel();
    super.dispose();
  }
}
