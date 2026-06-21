import 'package:flutter/foundation.dart';
import '../../data/api/api_client.dart';
import '../../data/dto/admin_models.dart';

class AdminViewModel extends ChangeNotifier {
  final ApiClient _apiClient;

  bool _isLoading = true;
  String _error = '';
  AdminHealth? _health;
  List<AdminEndpoint>? _endpoints;
  AdminConfig? _config;
  bool _isSavingConfig = false;

  AdminViewModel({required ApiClient apiClient}) : _apiClient = apiClient;

  bool get isLoading => _isLoading;
  String get error => _error;
  AdminHealth? get health => _health;
  List<AdminEndpoint>? get endpoints => _endpoints;
  AdminConfig? get config => _config;
  bool get isSavingConfig => _isSavingConfig;

  Future<void> loadData() async {
    _isLoading = true;
    _error = '';
    notifyListeners();

    final results = await Future.wait([
      _apiClient.adminHealth(),
      _apiClient.adminEndpoints(),
      _apiClient.adminGetConfig(),
    ]);

    _isLoading = false;
    _health = (results[0] != null)
        ? AdminHealth.fromJson(results[0]!)
        : null;
    _endpoints = (results[1]?['endpoints'] is List)
        ? (results[1]!['endpoints'] as List)
            .map((e) => AdminEndpoint.fromJson(e as Map<String, dynamic>))
            .toList()
        : null;
    _config = (results[2] != null)
        ? AdminConfig.fromJson(results[2]!)
        : null;

    if (_health == null && _config == null) {
      _error = 'Failed to load admin data.\nCheck server connectivity.';
    }
    notifyListeners();
  }

  Future<bool> saveConfig({
    required String subscriptionUrl,
    required String refreshInterval,
  }) async {
    _isSavingConfig = true;
    notifyListeners();

    final ok = await _apiClient.adminUpdateConfig(
      subscriptionUrl: subscriptionUrl,
      refreshInterval: refreshInterval,
    );

    _isSavingConfig = false;
    notifyListeners();

    if (ok) {
      await loadData();
    }
    return ok;
  }

  Future<bool> refreshConfigs() async {
    final ok = await _apiClient.adminRefreshConfigs();
    if (ok) {
      await Future.delayed(const Duration(seconds: 2));
      await loadData();
    }
    return ok;
  }

  Future<void> logout() async {
    await _apiClient.adminLogout();
  }
}
