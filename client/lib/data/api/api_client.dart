import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/vpn_config.dart';

class ApiClient {
  final Dio _dio;
  String? _baseUrl;
  SharedPreferences? _prefs;

  static String? _webUrl;
  static const String _defaultWebUrl = 'http://localhost:8080';
  static const String _baseUrlKey = 'server_url';
  static const String _adminTokenKey = 'admin_token';

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

  // -- Admin auth --

  Future<String?> getAdminToken() async {
    return _prefs?.getString(_adminTokenKey);
  }

  Future<void> _saveAdminToken(String token) async {
    await _prefs?.setString(_adminTokenKey, token);
  }

  Future<void> clearAdminToken() async {
    await _prefs?.remove(_adminTokenKey);
  }

  Map<String, dynamic> _authHeaders(String? token) {
    if (token == null) return {};
    return {'Authorization': 'Bearer $token'};
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
        await _saveAdminToken(token);
        return token;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.adminLogin error: $e');
      return null;
    }
  }

  Future<Map<String, dynamic>?> adminHealth() async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return null;

    try {
      final response = await _dio.get(
        '$url/api/admin/health',
        options: Options(headers: _authHeaders(token)),
      );
      if (response.statusCode == 200) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.adminHealth error: $e');
      return null;
    }
  }

  Future<Map<String, dynamic>?> adminEndpoints() async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return null;

    try {
      final response = await _dio.get(
        '$url/api/admin/endpoints',
        options: Options(headers: _authHeaders(token)),
      );
      if (response.statusCode == 200) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.adminEndpoints error: $e');
      return null;
    }
  }

  Future<Map<String, dynamic>?> adminGetConfig() async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return null;

    try {
      final response = await _dio.get(
        '$url/api/admin/config',
        options: Options(headers: _authHeaders(token)),
      );
      if (response.statusCode == 200) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.adminGetConfig error: $e');
      return null;
    }
  }

  Future<bool> adminUpdateConfig({
    String? subscriptionUrl,
    String? refreshInterval,
  }) async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return false;

    try {
      final data = <String, dynamic>{};
      if (subscriptionUrl != null) data['subscription_url'] = subscriptionUrl;
      if (refreshInterval != null) data['refresh_interval'] = refreshInterval;

      final response = await _dio.put(
        '$url/api/admin/config',
        data: data,
        options: Options(headers: _authHeaders(token)),
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminUpdateConfig error: $e');
      return false;
    }
  }

  Future<bool> adminRefreshConfigs() async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return false;

    try {
      final response = await _dio.post(
        '$url/api/admin/refresh-configs',
        options: Options(headers: _authHeaders(token)),
      );
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('ApiClient.adminRefreshConfigs error: $e');
      return false;
    }
  }

  Future<bool> adminLogout() async {
    final url = serverUrl;
    final token = await getAdminToken();
    if (url == null || token == null) return false;

    try {
      await _dio.post(
        '$url/api/admin/logout',
        options: Options(headers: _authHeaders(token)),
      );
    } catch (_) {}
    await clearAdminToken();
    return true;
  }

  // -- Public API methods --

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

  Future<Map<String, dynamic>?> checkForUpdate() async {
    final url = serverUrl;
    if (url == null) return null;

    try {
      final response = await _dio.get('$url/api/update');
      if (response.statusCode == 200) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      debugPrint('ApiClient.checkForUpdate error: $e');
      return null;
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

  String _normalizeUrl(String url) {
    return url.endsWith('/') ? url.substring(0, url.length - 1) : url;
  }
}
