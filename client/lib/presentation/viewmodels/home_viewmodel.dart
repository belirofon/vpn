import 'dart:async';
import 'package:flutter/foundation.dart';
import '../../data/api/api_client.dart';
import '../../domain/entities/vpn_config.dart';
import '../../domain/entities/warp_config.dart';
import '../../domain/services/vpn_service.dart';

enum HomeMode { warp, proxy }

enum ScreenState { disconnected, connecting, connected, error }

class HomeViewModel extends ChangeNotifier {
  final ApiClient _apiClient;
  final VpnService _vpnService;
  StreamSubscription<VpnConnectionState>? _stateSub;

  ScreenState _screenState = ScreenState.disconnected;
  VpnConfig? _activeConfig;
  WarpConfig? _activeWarpConfig;
  String? _errorMessage;
  HomeMode _selectedMode = HomeMode.proxy;

  List<VpnConfig> _proxyConfigs = [];
  WarpConfig? _warpConfig;
  bool _isLoadingConfigs = false;

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
  WarpConfig? get activeWarpConfig => _activeWarpConfig;
  String? get errorMessage => _errorMessage;
  HomeMode get selectedMode => _selectedMode;

  List<VpnConfig> get proxyConfigs => _proxyConfigs;
  WarpConfig? get warpConfig => _warpConfig;
  bool get isLoadingConfigs => _isLoadingConfigs;

  bool get isConnected => _screenState == ScreenState.connected;
  bool get isLoading => _screenState == ScreenState.connecting;

  void selectMode(HomeMode mode) {
    if (_selectedMode == mode) return;
    _selectedMode = mode;
    notifyListeners();
    loadConfigs();
  }

  Future<void> loadConfigs() async {
    _errorMessage = null;
    _isLoadingConfigs = true;
    notifyListeners();

    try {
      if (_selectedMode == HomeMode.proxy) {
        _proxyConfigs = await _apiClient.getConfigs();
      } else {
        _warpConfig = await _apiClient.getWarpConfig();
      }
    } catch (e) {
      debugPrint('HomeViewModel.loadConfigs error: $e');
    }

    _isLoadingConfigs = false;
    notifyListeners();
  }

  Future<void> connectToProxy(VpnConfig config) async {
    debugPrint('[HomeViewModel] connectToProxy — ${config.id} ${config.name}');
    _errorMessage = null;
    notifyListeners();

    try {
      await _vpnService.connect(config);
      debugPrint('[HomeViewModel] connectToProxy — OK, setting activeConfig');
      _activeConfig = config;
      _activeWarpConfig = null;
    } catch (e) {
      debugPrint('[HomeViewModel] connectToProxy — ERROR: $e');
      _errorMessage = e.toString();
      notifyListeners();
    }
  }

  Future<void> connectToWarp(WarpConfig config) async {
    debugPrint('[HomeViewModel] connectToWarp — ${config.endpoint}');
    _errorMessage = null;
    notifyListeners();

    try {
      await _vpnService.connectWarp(config);
      debugPrint('[HomeViewModel] connectToWarp — OK, setting activeWarpConfig');
      _activeWarpConfig = config;
      _activeConfig = null;
    } catch (e) {
      debugPrint('[HomeViewModel] connectToWarp — ERROR: $e');
      _errorMessage = e.toString();
      notifyListeners();
    }
  }

  Future<void> toggleConnection() async {
    debugPrint('[HomeViewModel] toggleConnection — currentState=$_screenState');
    if (_screenState == ScreenState.connected) {
      await _vpnService.disconnect();
      return;
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
    debugPrint('[HomeViewModel] _onVpnStateChanged: $state');
    _screenState = _vpnStateToScreen(state);
    if (state == VpnConnectionState.disconnected) {
      _activeConfig = null;
      _activeWarpConfig = null;
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
