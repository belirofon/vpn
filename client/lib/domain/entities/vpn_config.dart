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

  const VpnConfig({
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

  @override
  String toString() =>
      'VpnConfig($name, $protocol://$server:$port, ${latencyMs}ms, $country)';
}
