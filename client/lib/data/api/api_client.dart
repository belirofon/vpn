import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/vpn_config.dart';

class ApiClient {
  final Dio _dio;
  String? _baseUrl;
  SharedPreferences? _prefs;

  static String? _webUrl;
  // Default URL is always localhost (for local dev with mock server).
  // In production, SERVER_URL MUST be set via --dart-define at build time:
  //   flutter build apk --dart-define=SERVER_URL=https://your-domain.com:8443
  // Or changed at runtime via long-press debug menu (persisted to SharedPreferences).
  static const String _defaultWebUrl = 'http://localhost:8080';
  static const String _baseUrlKey = 'server_url';

  ApiClient(this._dio);

  void setBaseUrl(String url) {
    _baseUrl = _normalizeUrl(url);
    if (kIsWeb) {
      _webUrl = _baseUrl;
    }
  }

  Future<void> init(SharedPreferences prefs) async {
    _prefs = prefs;
    final saved = prefs.getString(_baseUrlKey);
    if (saved != null) {
      _baseUrl = saved;
    }
  }

  Future<void> saveServerUrl(String url) async {
    setBaseUrl(url);
    await _prefs?.setString(_baseUrlKey, url);
  }

  String? get serverUrl {
    if (kIsWeb) {
      return _webUrl ?? _defaultWebUrl;
    }
    return _baseUrl ?? _defaultWebUrl;
  }

  // -- API methods --

  Future<VpnConfig?> getBestConfig() async {
    final url = serverUrl;
    if (url == null) return null;

    try {
      final response = await _dio.get('$url/api/best-config');
      if (response.statusCode == 200 && response.data['config'] != null) {
        return VpnConfig.fromJson(response.data['config'] as Map<String, dynamic>);
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.getBestConfig error: $e');
      return null;
    }
  }

  Future<List<VpnConfig>> getConfigs() async {
    final url = serverUrl;
    if (url == null) return [];

    try {
      final response = await _dio.get('$url/api/configs');
      if (response.statusCode == 200 && response.data['configs'] is List) {
        final list = response.data['configs'] as List;
        return list
            .map((e) => VpnConfig.fromJson(e as Map<String, dynamic>))
            .toList();
      }
      return [];
    } catch (e) {
      debugPrint('ApiClient.getConfigs error: $e');
      return [];
    }
  }

  Future<bool> healthCheck() async {
    final url = serverUrl;
    if (url == null) return false;

    try {
      final response = await _dio.get('$url/health');
      return response.statusCode == 200;
    } catch (e) {
      return false;
    }
  }

  // -- Admin API methods --

  Future<String?> adminLogin(String email, String password) async {
    final url = serverUrl;
    if (url == null) return null;

    try {
      final response = await _dio.post(
        '$url/api/admin/login',
        data: {'email': email, 'password': password},
      );
      if (response.statusCode == 200 && response.data['token'] != null) {
        final token = response.data['token'] as String;
        await _prefs?.setString('admin_token', token);
        return token;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.adminLogin error: $e');
      return null;
    }
  }

  Future<Map<String, dynamic>?> adminHealth() async {
    return _adminGet('/api/admin/health');
  }

  Future<Map<String, dynamic>?> adminEndpoints() async {
    return _adminGet('/api/admin/endpoints');
  }

  Future<Map<String, dynamic>?> adminGetConfig() async {
    return _adminGet('/api/admin/config');
  }

  Future<bool> adminUpdateConfig({
    String? subscriptionUrl,
    String? refreshInterval,
  }) async {
    final url = serverUrl;
    final token = _prefs?.getString('admin_token');
    if (url == null || token == null) return false;

    try {
      final body = <String, dynamic>{};
      if (subscriptionUrl != null) body['subscription_url'] = subscriptionUrl;
      if (refreshInterval != null) body['refresh_interval'] = refreshInterval;
      final response = await _dio.put(
        '$url/api/admin/config',
        data: body,
        options: Options(headers: {'Authorization': 'Bearer $token'}),
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminUpdateConfig error: $e');
      return false;
    }
  }

  Future<bool> adminRefreshConfigs() async {
    final url = serverUrl;
    final token = _prefs?.getString('admin_token');
    if (url == null || token == null) return false;

    try {
      final response = await _dio.post(
        '$url/api/admin/refresh-configs',
        options: Options(headers: {'Authorization': 'Bearer $token'}),
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminRefreshConfigs error: $e');
      return false;
    }
  }

  Future<bool> adminLogout() async {
    final url = serverUrl;
    final token = _prefs?.getString('admin_token');
    if (url == null || token == null) return false;

    try {
      await _dio.post(
        '$url/api/admin/logout',
        options: Options(headers: {'Authorization': 'Bearer $token'}),
      );
      await _prefs?.remove('admin_token');
      return true;
    } catch (e) {
      debugPrint('ApiClient.adminLogout error: $e');
      return false;
    }
  }

  Future<Map<String, dynamic>?> _adminGet(String path) async {
    final url = serverUrl;
    final token = _prefs?.getString('admin_token');
    if (url == null || token == null) return null;

    try {
      final response = await _dio.get(
        '$url$path',
        options: Options(headers: {'Authorization': 'Bearer $token'}),
      );
      if (response.statusCode == 200) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient._adminGet($path) error: $e');
      return null;
    }
  }

  String _normalizeUrl(String url) {
    return url.endsWith('/') ? url.substring(0, url.length - 1) : url;
  }
}
