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
  final String? host;
  final String? path;
  final String? sni;
  final String? fp;
  final String? pbk;
  final String? sid;
  final String? flow;

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
    this.host,
    this.path,
    this.sni,
    this.fp,
    this.pbk,
    this.sid,
    this.flow,
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
      host: json['host'] as String?,
      path: json['path'] as String?,
      sni: json['sni'] as String?,
      fp: json['fp'] as String?,
      pbk: json['pbk'] as String?,
      sid: json['sid'] as String?,
      flow: json['flow'] as String?,
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
      if (host != null) 'host': host,
      if (path != null) 'path': path,
      if (sni != null) 'sni': sni,
      if (fp != null) 'fp': fp,
      if (pbk != null) 'pbk': pbk,
      if (sid != null) 'sid': sid,
      if (flow != null) 'flow': flow,
    };
  }

  @override
  String toString() =>
      'VpnConfig($name, $protocol://$server:$port, ${latencyMs}ms, $country)';
}
