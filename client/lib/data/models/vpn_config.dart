class VpnConfig {
  final String id;
  final String name;
  final String server;
  final int port;
  final String protocol;
  final String? uuid;
  final String? password;
  final String? tls;
  final String? network;
  final int latencyMs;
  final String country;
  final String? rawLink;

  VpnConfig({
    required this.id,
    required this.name,
    required this.server,
    required this.port,
    required this.protocol,
    this.uuid,
    this.password,
    this.tls,
    this.network,
    this.latencyMs = 0,
    this.country = '',
    this.rawLink,
  });

  /// Parse a raw proxy link (vless://, vmess://, etc.) into a VpnConfig.
  factory VpnConfig.fromRawLink(String link) {
    final uri = Uri.parse(link);
    final protocol = uri.scheme; // vless, vmess, trojan, etc.
    final uuid = uri.userInfo.isNotEmpty ? uri.userInfo : null;
    final query = uri.queryParameters;
    final fragment = uri.fragment;

    return VpnConfig(
      id: link.hashCode.toString(),
      name: fragment.isNotEmpty
          ? Uri.decodeComponent(fragment)
          : 'Manual (${uri.host}:${uri.port})',
      server: uri.host,
      port: uri.port,
      protocol: protocol,
      uuid: uuid,
      network: query['type'] ?? 'tcp',
      tls: query['security'] == 'tls' ? 'tls' : null,
      rawLink: link,
    );
  }

  factory VpnConfig.fromJson(Map<String, dynamic> json) {
    return VpnConfig(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      server: json['server'] as String? ?? '',
      port: json['port'] as int? ?? 0,
      protocol: json['protocol'] as String? ?? '',
      uuid: json['uuid'] as String?,
      password: json['password'] as String?,
      tls: json['tls'] as String?,
      network: json['network'] as String?,
      latencyMs: json['latency_ms'] as int? ?? 0,
      country: json['country'] as String? ?? '',
      rawLink: json['raw_link'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'server': server,
      'port': port,
      'protocol': protocol,
      if (uuid != null) 'uuid': uuid,
      if (password != null) 'password': password,
      if (tls != null) 'tls': tls,
      if (network != null) 'network': network,
      'latency_ms': latencyMs,
      'country': country,
      if (rawLink != null) 'raw_link': rawLink,
    };
  }

  @override
  String toString() =>
      'VpnConfig($name, $protocol://$server:$port, ${latencyMs}ms, $country)';
}
