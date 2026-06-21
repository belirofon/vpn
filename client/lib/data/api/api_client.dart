import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../../domain/entities/vpn_config.dart';
import '../../domain/repositories/http_client.dart';
import '../datasources/dio_http_client.dart';
import '../dto/vpn_config_dto.dart';

class ApiClient {
  late final HttpClient _http;
  String? _baseUrl;
  SharedPreferences? _prefs;

  static String? _webUrl;
  static const String _defaultWebUrl = 'http://localhost:8080';
  static const String _baseUrlKey = 'server_url';

  ApiClient() {
    _http = DioHttpClient();
  }

  void setBaseUrl(String url) {
    _baseUrl = _normalizeUrl(url);
    _http.updateBaseUrl(_baseUrl!);
    if (kIsWeb) {
      _webUrl = _baseUrl;
    }
  }

  Future<void> init(SharedPreferences prefs) async {
    _prefs = prefs;
    final saved = prefs.getString(_baseUrlKey);
    if (saved != null) {
      setBaseUrl(saved);
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
    try {
      final response = await _http.get('/api/best-config');
      if (response.statusCode == 200 && response.data?['config'] != null) {
        return VpnConfigDto.fromJson(
            response.data!['config'] as Map<String, dynamic>);
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.getBestConfig error: $e');
      return null;
    }
  }

  Future<List<VpnConfig>> getConfigs() async {
    try {
      final response = await _http.get('/api/configs');
      if (response.statusCode == 200 && response.data?['configs'] is List) {
        final list = response.data!['configs'] as List;
        return list
            .map((e) =>
                VpnConfigDto.fromJson(e as Map<String, dynamic>))
            .toList();
      }
      return [];
    } catch (e) {
      debugPrint('ApiClient.getConfigs error: $e');
      return [];
    }
  }

  Future<bool> healthCheck() async {
    try {
      final response = await _http.get('/health');
      return response.statusCode == 200;
    } catch (e) {
      return false;
    }
  }

  // -- Admin API methods --

  Future<String?> adminLogin(String email, String password) async {
    try {
      final response = await _http.post(
        '/api/admin/login',
        data: {'email': email, 'password': password},
      );
      if (response.statusCode == 200 && response.data?['token'] != null) {
        final token = response.data!['token'] as String;
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
    final token = _prefs?.getString('admin_token');
    if (token == null) return false;

    try {
      final body = <String, dynamic>{};
      if (subscriptionUrl != null) body['subscription_url'] = subscriptionUrl;
      if (refreshInterval != null) body['refresh_interval'] = refreshInterval;
      final response = await _http.put(
        '/api/admin/config',
        data: body,
        headers: {'Authorization': 'Bearer $token'},
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminUpdateConfig error: $e');
      return false;
    }
  }

  Future<bool> adminRefreshConfigs() async {
    final token = _prefs?.getString('admin_token');
    if (token == null) return false;

    try {
      final response = await _http.post(
        '/api/admin/refresh-configs',
        headers: {'Authorization': 'Bearer $token'},
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminRefreshConfigs error: $e');
      return false;
    }
  }

  Future<bool> adminLogout() async {
    final token = _prefs?.getString('admin_token');
    if (token == null) return false;

    try {
      await _http.post(
        '/api/admin/logout',
        headers: {'Authorization': 'Bearer $token'},
      );
      await _prefs?.remove('admin_token');
      return true;
    } catch (e) {
      debugPrint('ApiClient.adminLogout error: $e');
      return false;
    }
  }

  Future<Map<String, dynamic>?> _adminGet(String path) async {
    final token = _prefs?.getString('admin_token');
    if (token == null) return null;

    try {
      final response = await _http.get(
        path,
        headers: {'Authorization': 'Bearer $token'},
      );
      if (response.statusCode == 200) {
        return response.data;
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
