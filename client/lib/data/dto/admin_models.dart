import '../../domain/entities/warp_config.dart';

class AdminHealth {
  final String status;
  final String? message;
  final int configsTested;
  final String uptime;
  final String subscriptionUrl;
  final String refreshInterval;

  const AdminHealth({
    required this.status,
    this.message,
    this.configsTested = 0,
    this.uptime = '',
    this.subscriptionUrl = '',
    this.refreshInterval = '',
  });

  factory AdminHealth.fromJson(Map<String, dynamic> json) {
    return AdminHealth(
      status: json['status'] as String? ?? '',
      message: json['message'] as String?,
      configsTested: json['configs_tested'] as int? ?? 0,
      uptime: json['uptime'] as String? ?? '',
      subscriptionUrl: json['subscription_url'] as String? ?? '',
      refreshInterval: json['refresh_interval'] as String? ?? '',
    );
  }
}

class AdminConfig {
  final String subscriptionUrl;
  final String refreshInterval;
  final String pingTimeout;
  final bool mockConfigs;
  final bool skipVerifyTls;
  final String corsOrigins;

  const AdminConfig({
    required this.subscriptionUrl,
    required this.refreshInterval,
    required this.pingTimeout,
    required this.mockConfigs,
    required this.skipVerifyTls,
    required this.corsOrigins,
  });

  factory AdminConfig.fromJson(Map<String, dynamic> json) {
    return AdminConfig(
      subscriptionUrl: json['subscription_url'] as String? ?? '',
      refreshInterval: json['refresh_interval'] as String? ?? '',
      pingTimeout: json['ping_timeout'] as String? ?? '',
      mockConfigs: json['mock_configs'] as bool? ?? false,
      skipVerifyTls: json['skip_verify_tls'] as bool? ?? true,
      corsOrigins: json['cors_origins'] as String? ?? '',
    );
  }
}

class AdminEndpoint {
  final String method;
  final String path;

  const AdminEndpoint({required this.method, required this.path});

  factory AdminEndpoint.fromJson(Map<String, dynamic> json) {
    return AdminEndpoint(
      method: json['method'] as String? ?? '',
      path: json['path'] as String? ?? '',
    );
  }

  int methodColorValue() {
    return switch (method) {
      'GET' => 0xFF4CAF50,
      'POST' => 0xFFFF9800,
      'PUT' => 0xFF2196F3,
      'DELETE' => 0xFFF44336,
      'PATCH' => 0xFF9C27B0,
      _ => 0xFF9E9E9E,
    };
  }
}

class AdminWarpStatus {
  final bool available;
  final WarpConfig? config;

  const AdminWarpStatus({required this.available, this.config});

  factory AdminWarpStatus.fromJson(Map<String, dynamic> json) {
    return AdminWarpStatus(
      available: json['available'] as bool? ?? false,
      config: json['config'] != null
          ? WarpConfig.fromJson(json['config'] as Map<String, dynamic>)
          : null,
    );
  }
}
