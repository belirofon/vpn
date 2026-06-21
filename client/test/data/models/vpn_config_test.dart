import 'package:flutter_test/flutter_test.dart';
import 'package:vpn_client/data/dto/vpn_config_dto.dart';
import 'package:vpn_client/domain/entities/vpn_config.dart';

void main() {
  group('VpnConfigDto.fromJson', () {
    test('parses all fields correctly', () {
      final json = {
        'id': 'server.com:443',
        'name': 'NL-01',
        'server': '192.168.1.1',
        'port': 443,
        'protocol': 'vless',
        'uuid': '550e8400-e29b-41d4-a716-446655440000',
        'tls': 'tls',
        'network': 'ws',
        'latency_ms': 87,
        'country': 'NL',
        'raw_link': 'vless://...',
      };

      final config = VpnConfigDto.fromJson(json);

      expect(config.id, 'server.com:443');
      expect(config.name, 'NL-01');
      expect(config.server, '192.168.1.1');
      expect(config.port, 443);
      expect(config.protocol, 'vless');
      expect(config.uuid, '550e8400-e29b-41d4-a716-446655440000');
      expect(config.tls, 'tls');
      expect(config.network, 'ws');
      expect(config.latencyMs, 87);
      expect(config.country, 'NL');
      expect(config.rawLink, 'vless://...');
    });

    test('handles missing optional fields', () {
      final json = {
        'id': 's.com:80',
        'name': 'test',
        'server': 's.com',
        'port': 80,
        'protocol': 'vless',
        'latency_ms': 0,
        'country': '',
      };

      final config = VpnConfigDto.fromJson(json);

      expect(config.uuid, isNull);
      expect(config.password, isNull);
      expect(config.tls, isNull);
      expect(config.network, isNull);
      expect(config.rawLink, isNull);
      expect(config.latencyMs, 0);
      expect(config.country, '');
    });

    test('handles null values', () {
      final json = {
        'id': null,
        'name': null,
        'server': null,
        'port': null,
        'protocol': null,
        'latency_ms': null,
        'country': null,
      };

      final config = VpnConfigDto.fromJson(json);

      expect(config.id, '');
      expect(config.name, '');
      expect(config.server, '');
      expect(config.port, 0);
      expect(config.protocol, '');
      expect(config.latencyMs, 0);
      expect(config.country, '');
    });
  });

  group('VpnConfigDto.toJson', () {
    test('serializes all fields correctly', () {
      const config = VpnConfig(
        id: 'server.com:443',
        name: 'DE-01',
        server: '192.168.1.1',
        port: 443,
        protocol: 'vless',
        uuid: 'uuid-here',
        password: 'pass',
        tls: 'tls',
        network: 'tcp',
        latencyMs: 45,
        country: 'DE',
        rawLink: 'vless://...',
      );

      final json = VpnConfigDto.toJson(config);

      expect(json['id'], 'server.com:443');
      expect(json['name'], 'DE-01');
      expect(json['server'], '192.168.1.1');
      expect(json['port'], 443);
      expect(json['uuid'], 'uuid-here');
      expect(json['password'], 'pass');
      expect(json['tls'], 'tls');
      expect(json['network'], 'tcp');
      expect(json['latency_ms'], 45);
      expect(json['country'], 'DE');
      expect(json['raw_link'], 'vless://...');
    });

    test('omits null fields', () {
      const config = VpnConfig(
        id: 's.com:80',
        name: 'test',
        server: 's.com',
        port: 80,
        protocol: 'http',
      );

      final json = VpnConfigDto.toJson(config);

      expect(json.containsKey('uuid'), isFalse);
      expect(json.containsKey('password'), isFalse);
      expect(json.containsKey('tls'), isFalse);
      expect(json.containsKey('network'), isFalse);
      expect(json.containsKey('raw_link'), isFalse);
    });
  });

  group('VpnConfigDto.fromRawLink', () {
    test('parses vless link', () {
      const link = 'vless://uuid@server.com:443?security=tls&type=ws#NL-01';
      final config = VpnConfigDto.fromRawLink(link);

      expect(config.protocol, 'vless');
      expect(config.uuid, 'uuid');
      expect(config.server, 'server.com');
      expect(config.port, 443);
      expect(config.network, 'ws');
      expect(config.tls, 'tls');
      expect(config.rawLink, link);
    });

    test('parses link without fragment for name', () {
      const link = 'vless://uuid@server.com:443';
      final config = VpnConfigDto.fromRawLink(link);

      expect(config.name, 'Manual (server.com:443)');
      expect(config.id, isNot(throwsException));
    });

    test('parses link with empty query', () {
      const link = 'vmess://uuid@server.com:443';
      final config = VpnConfigDto.fromRawLink(link);

      expect(config.protocol, 'vmess');
      expect(config.uuid, 'uuid');
      expect(config.server, 'server.com');
      expect(config.port, 443);
    });
  });

  group('VpnConfig.toString', () {
    test('formats correctly with all fields', () {
      const config = VpnConfig(
        id: 's.com:443',
        name: 'DE-01',
        server: 's.com',
        port: 443,
        protocol: 'vless',
        latencyMs: 45,
        country: 'DE',
      );

      final str = config.toString();
      expect(str, contains('DE-01'));
      expect(str, contains('vless://s.com:443'));
      expect(str, contains('45ms'));
      expect(str, contains('DE'));
    });
  });
}
