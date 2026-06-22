import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:path_provider/path_provider.dart';
import 'package:pub_semver/pub_semver.dart';
import '../../data/api/api_client.dart';

/// Whether a newer version is available and how to treat it.
enum UpdateStatus { upToDate, suggested, required_ }

/// Server response with version info.
class AppVersionInfo {
  final Version latest;
  final Version minimum;
  final int buildNumber;
  final String changelog;

  AppVersionInfo({
    required this.latest,
    required this.minimum,
    required this.buildNumber,
    required this.changelog,
  });

  factory AppVersionInfo.fromJson(Map<String, dynamic> json) {
    return AppVersionInfo(
      latest: Version.parse(json['version'] as String? ?? '0.0.0'),
      minimum: Version.parse(json['min_version'] as String? ?? '0.0.0'),
      buildNumber: json['build_number'] as int? ?? 1,
      changelog: json['changelog'] as String? ?? '',
    );
  }
}

/// Result of checking for updates.
class UpdateCheckResult {
  final UpdateStatus status;
  final AppVersionInfo info;

  const UpdateCheckResult({required this.status, required this.info});
}

/// Handles checking for app updates, downloading APK, and triggering install.
class UpdateService {
  final ApiClient _apiClient;
  final Dio _dio;
  static const _channel = MethodChannel('vpn_client/install');

  UpdateService(this._apiClient)
      : _dio = Dio(
          BaseOptions(
            connectTimeout: const Duration(seconds: 15),
            receiveTimeout: const Duration(seconds: 120),
            sendTimeout: const Duration(seconds: 15),
          ),
        );

  /// Fetches version info from server and decides update status.
  Future<UpdateCheckResult?> check() async {
    final data = await _apiClient.checkForUpdate();
    if (data == null) return null;

    final info = AppVersionInfo.fromJson(data);
    final pkg = await PackageInfo.fromPlatform();
    final current = Version.parse(pkg.version);

    late final UpdateStatus status;
    if (current < info.minimum) {
      status = UpdateStatus.required_;
    } else if (current < info.latest) {
      status = UpdateStatus.suggested;
    } else {
      status = UpdateStatus.upToDate;
    }

    return UpdateCheckResult(status: status, info: info);
  }

  /// Downloads APK from the same server the client is configured to use.
  /// Returns the local file path on success.
  Future<String> download({
    void Function(int received, int total)? onProgress,
  }) async {
    final dir = await getApplicationDocumentsDirectory();
    final filePath = '${dir.path}/vpn-client-android.apk';
    final url = '${_apiClient.serverUrl ?? 'http://localhost:8080'}/api/update/download';

    await _dio.download(
      url,
      filePath,
      onReceiveProgress: onProgress,
    );

    return filePath;
  }

  /// Triggers Android package installer via FileProvider.
  /// Returns true if the installer was launched successfully.
  /// Throws [PlatformException] with code `PERMISSION_REQUIRED` if the user
  /// must first grant the "Install unknown apps" permission in Settings.
  Future<bool> install(String filePath) async {
    if (kIsWeb || !Platform.isAndroid) return false;

    try {
      final result = await _channel.invokeMethod<bool>('installApk', {'path': filePath});
      return result ?? false;
    } on MissingPluginException {
      return false;
    } on PlatformException catch (e) {
      if (e.code == 'PERMISSION_REQUIRED') rethrow;
      debugPrint('UpdateService.install (platform): $e');
      return false;
    } catch (e) {
      debugPrint('UpdateService.install: $e');
      return false;
    }
  }
}
