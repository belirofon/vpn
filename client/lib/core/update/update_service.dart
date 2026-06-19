import 'dart:async';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:path_provider/path_provider.dart';
import '../../data/api/api_client.dart';

class UpdateInfo {
  final String version;
  final int buildNumber;
  final String downloadUrl;
  final String changelog;

  UpdateInfo({
    required this.version,
    required this.buildNumber,
    required this.downloadUrl,
    required this.changelog,
  });

  factory UpdateInfo.fromJson(Map<String, dynamic> json) {
    return UpdateInfo(
      version: json['version'] as String? ?? '',
      buildNumber: json['build_number'] as int? ?? 0,
      downloadUrl: json['download_url'] as String? ?? '',
      changelog: json['changelog'] as String? ?? '',
    );
  }
}

class UpdateService {
  final ApiClient _apiClient;
  final Dio _dio;

  UpdateService(this._apiClient) : _dio = Dio();

  static const _channel = MethodChannel('vpn_client/install');

  Future<UpdateInfo?> checkForUpdate() async {
    final data = await _apiClient.checkForUpdate();
    if (data == null) return null;
    return UpdateInfo.fromJson(data);
  }

  bool isNewer(String currentVersion, UpdateInfo update) {
    final current = currentVersion.split('.').map(int.tryParse).toList();
    final latest = update.version.split('.').map(int.tryParse).toList();
    if (current.any((e) => e == null) || latest.any((e) => e == null)) {
      return false;
    }
    for (int i = 0; i < latest.length; i++) {
      final c = i < current.length ? current[i]! : 0;
      final l = latest[i]!;
      if (l > c) return true;
      if (l < c) return false;
    }
    return false;
  }

  Future<String> downloadApk({
    required String url,
    void Function(int received, int total)? onProgress,
  }) async {
    final dir = await getApplicationDocumentsDirectory();
    final filePath = '${dir.path}/vpn-client-android.apk';

    await _dio.download(
      url,
      filePath,
      onReceiveProgress: onProgress,
    );

    return filePath;
  }

  Future<bool> installApk(String filePath) async {
    if (!kIsWeb && Platform.isAndroid) {
      try {
        await _channel.invokeMethod('installApk', {'path': filePath});
        return true;
      } catch (e) {
        debugPrint('UpdateService.installApk error: $e');
        return false;
      }
    }
    return false;
  }
}
