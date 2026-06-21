class WarpConfig {
  final String protocol;
  final String privateKey;
  final String addressV4;
  final String addressV6;
  final String dnsServers;
  final String serverPublicKey;
  final String endpoint;
  final String? clientId;
  final int latencyMs;

  const WarpConfig({
    required this.protocol,
    required this.privateKey,
    required this.addressV4,
    required this.addressV6,
    required this.dnsServers,
    required this.serverPublicKey,
    required this.endpoint,
    this.clientId,
    this.latencyMs = 0,
  });

  factory WarpConfig.fromJson(Map<String, dynamic> json) {
    return WarpConfig(
      protocol: json['protocol'] as String? ?? '',
      privateKey: json['private_key'] as String? ?? '',
      addressV4: json['address_v4'] as String? ?? '',
      addressV6: json['address_v6'] as String? ?? '',
      dnsServers: json['dns'] as String? ?? '',
      serverPublicKey: json['server_public_key'] as String? ?? '',
      endpoint: json['endpoint'] as String? ?? '',
      clientId: json['client_id'] as String?,
      latencyMs: json['latency_ms'] as int? ?? 0,
    );
  }

  String get displayName {
    if (clientId != null && clientId!.isNotEmpty) {
      return 'WARP $clientId';
    }
    return 'WARP $endpoint';
  }

  @override
  String toString() =>
      'WarpConfig($displayName, $endpoint, ${latencyMs}ms)';
}
