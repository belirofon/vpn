import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/vpn_config.dart';

class ApiClient {
  final Dio _dio;
  String? _baseUrl;
  SharedPreferences? _prefs;

  static String? _webUrl;
  // Debug: localhost (for local dev with mock server)
  // Release: production server (overridable via --dart-define=SERVER_URL or debug menu)
  static const String _defaultWebUrl = kReleaseMode
      ? 'https://belirofon-vpn.duckdns.org:8443'
      : 'http://localhost:8080';
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

  String _normalizeUrl(String url) {
    return url.endsWith('/') ? url.substring(0, url.length - 1) : url;
  }
}
