import '../../domain/entities/vpn_config.dart';

class VpnConfigDto {
  static VpnConfig fromJson(Map<String, dynamic> json) {
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
      singboxConfig: json['singbox_config'] as Map<String, dynamic>?,
    );
  }

  static Map<String, dynamic> toJson(VpnConfig config) {
    return {
      'id': config.id,
      'name': config.name,
      'server': config.server,
      'port': config.port,
      'protocol': config.protocol,
      if (config.uuid != null) 'uuid': config.uuid,
      if (config.password != null) 'password': config.password,
      if (config.tls != null) 'tls': config.tls,
      if (config.network != null) 'network': config.network,
      'latency_ms': config.latencyMs,
      'country': config.country,
      if (config.rawLink != null) 'raw_link': config.rawLink,
      if (config.singboxConfig != null) 'singbox_config': config.singboxConfig,
    };
  }

  static VpnConfig fromRawLink(String link) {
    final uri = Uri.parse(link);
    final protocol = uri.scheme;
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
}
