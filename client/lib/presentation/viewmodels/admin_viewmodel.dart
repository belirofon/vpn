import 'dart:convert';
import 'package:flutter/foundation.dart';
import '../../data/api/api_client.dart';
import '../../data/dto/admin_models.dart';
import '../../domain/entities/warp_config.dart';

class AdminViewModel extends ChangeNotifier {
  final ApiClient _apiClient;

  bool _isLoading = true;
  String _error = '';
  AdminHealth? _health;
  List<AdminEndpoint>? _endpoints;
  AdminConfig? _config;
  bool _isSavingConfig = false;
  AdminWarpStatus? _warpStatus;
  bool _isWarpLoading = false;
  bool _isWarpGenerating = false;
  bool _isScanning = false;
  String _scanResult = '';
  List<Map<String, dynamic>>? _bestConfigs;
  bool _isBestConfigsLoading = false;
  bool _isImporting = false;
  String _importResult = '';

  AdminViewModel({required ApiClient apiClient}) : _apiClient = apiClient;

  bool get isLoading => _isLoading;
  String get error => _error;
  AdminHealth? get health => _health;
  List<AdminEndpoint>? get endpoints => _endpoints;
  AdminConfig? get config => _config;
  bool get isSavingConfig => _isSavingConfig;
  AdminWarpStatus? get warpStatus => _warpStatus;
  bool get isWarpLoading => _isWarpLoading;
  bool get isWarpGenerating => _isWarpGenerating;
  bool get isScanning => _isScanning;
  String get scanResult => _scanResult;
  List<Map<String, dynamic>>? get bestConfigs => _bestConfigs;
  bool get isBestConfigsLoading => _isBestConfigsLoading;
  bool get isImporting => _isImporting;
  String get importResult => _importResult;

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

  Future<void> loadWarp() async {
    _isWarpLoading = true;
    notifyListeners();

    final result = await _apiClient.adminGetWarp();
    if (result != null) {
      _warpStatus = AdminWarpStatus.fromJson(result);
    }

    _isWarpLoading = false;
    notifyListeners();
  }

  Future<WarpConfig?> generateWarp() async {
    _isWarpGenerating = true;
    notifyListeners();

    final result = await _apiClient.adminGenerateWarp();
    if (result != null && result['config'] != null) {
      final config = WarpConfig.fromJson(
          result['config'] as Map<String, dynamic>);
      _warpStatus = AdminWarpStatus(available: true, config: config);
      _isWarpGenerating = false;
      notifyListeners();
      return config;
    }

    // Reload to get fresh state
    await loadWarp();
    _isWarpGenerating = false;
    notifyListeners();
    return null;
  }

  /// Processes scanned QR text — either a URL (fetch) or JSON (parse) or proxy link — and posts to server.
  Future<String> processScannedText(String raw) async {
    _isScanning = true;
    _scanResult = '';
    notifyListeners();

    try {
      final trimmed = raw.trim();
      Map<String, dynamic> config;

      if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
        final response = await _apiClient.fetchJson(trimmed);
        if (response == null) {
          _scanResult = 'Failed to fetch config from URL';
          return _scanResult;
        }
        config = response;
      } else if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
        config = _parseJson(trimmed);
      } else {
        config = {
          'raw_link': trimmed,
          'id': trimmed.hashCode.toString(),
          'name': 'Scanned (${_hostFromLink(trimmed)})',
          'server': _hostFromLink(trimmed),
          'port': _portFromLink(trimmed),
          'protocol': trimmed.split('://').first,
        };
      }

      final ok = await _apiClient.adminPostBestConfig(config);
      if (ok) {
        _scanResult = 'Config added successfully';
        loadBestConfigs();
      } else {
        _scanResult = 'Failed to add config to server';
      }
    } catch (e) {
      _scanResult = 'Error: $e';
    } finally {
      _isScanning = false;
      notifyListeners();
    }
    return _scanResult;
  }

  Map<String, dynamic> _parseJson(String raw) {
    try {
      // ignore: avoid_dynamic_calls
      final parsed = jsonDecode(raw) as Map<String, dynamic>;
      return parsed;
    } catch (_) {
      return {'raw_link': raw, 'id': raw.hashCode.toString(), 'name': 'Scanned config'};
    }
  }

  String _hostFromLink(String link) {
    try {
      final uri = Uri.parse(link);
      return uri.host;
    } catch (_) {
      return 'unknown';
    }
  }

  int _portFromLink(String link) {
    try {
      final uri = Uri.parse(link);
      return uri.port;
    } catch (_) {
      return 0;
    }
  }

  Future<bool> deleteWarp() async {
    final ok = await _apiClient.adminDeleteWarp();
    if (ok) {
      _warpStatus = const AdminWarpStatus(available: false);
      notifyListeners();
    }
    return ok;
  }

  // -- Best Configs Management --

  Future<void> loadBestConfigs() async {
    _isBestConfigsLoading = true;
    notifyListeners();

    _bestConfigs = await _apiClient.adminGetBestConfigs();

    _isBestConfigsLoading = false;
    notifyListeners();
  }

  Future<bool> deleteBestConfig(String id) async {
    final ok = await _apiClient.adminDeleteBestConfig(id);
    if (ok) {
      await loadBestConfigs();
    }
    return ok;
  }

  /// Imports configs from a subscription URL. Returns the server response message.
  Future<String> importFromUrl(String url) async {
    _isImporting = true;
    _importResult = '';
    notifyListeners();

    try {
      final result = await _apiClient.adminImportBestConfigs(url: url);
      if (result != null) {
        if (result['error'] != null) {
          _importResult = result['message'] as String? ?? 'Import failed';
        } else {
          final added = result['added'] ?? 0;
          _importResult = added is int && added > 0
              ? 'Imported $added configs'
              : 'No configs found at URL';
        }
      } else {
        _importResult = 'Failed to import from URL';
      }
    } catch (e) {
      _importResult = 'Error: $e';
    }

    _isImporting = false;
    notifyListeners();
    await loadBestConfigs();
    return _importResult;
  }

  /// Imports configs from raw proxy links (one per line).
  Future<String> importFromRawLinks(String rawText) async {
    _isImporting = true;
    _importResult = '';
    notifyListeners();

    try {
      final links = rawText
          .split('\n')
          .map((l) => l.trim())
          .where((l) => l.isNotEmpty)
          .toList();

      if (links.isEmpty) {
        _importResult = 'No links provided';
        _isImporting = false;
        notifyListeners();
        return _importResult;
      }

      final result = await _apiClient.adminImportBestConfigs(rawLinks: links);
      if (result != null) {
        final added = result['added'] ?? 0;
        _importResult = added is int && added > 0
            ? 'Imported $added configs'
            : 'No valid configs found';
      } else {
        _importResult = 'Failed to import configs';
      }
    } catch (e) {
      _importResult = 'Error: $e';
    }

    _isImporting = false;
    notifyListeners();
    await loadBestConfigs();
    return _importResult;
  }

  /// Imports configs from a JSON array of config objects.
  Future<String> importFromJson(String jsonText) async {
    _isImporting = true;
    _importResult = '';
    notifyListeners();

    try {
      final parsed = jsonDecode(jsonText);
      List<Map<String, dynamic>> configs;

      if (parsed is List) {
        configs = parsed.cast<Map<String, dynamic>>();
      } else if (parsed is Map<String, dynamic>) {
        configs = [parsed];
      } else {
        _importResult = 'Invalid JSON: expected object or array';
        _isImporting = false;
        notifyListeners();
        return _importResult;
      }

      final result = await _apiClient.adminImportBestConfigs(configs: configs);
      if (result != null) {
        final added = result['added'] ?? 0;
        _importResult = added is int && added > 0
            ? 'Imported $added configs'
            : 'No valid configs in JSON';
      } else {
        _importResult = 'Failed to import configs';
      }
    } catch (e) {
      _importResult = 'Error: $e';
    }

    _isImporting = false;
    notifyListeners();
    await loadBestConfigs();
    return _importResult;
  }
}
