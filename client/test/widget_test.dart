import 'dart:async';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vpn_client/data/api/api_client.dart';
import 'package:vpn_client/data/models/vpn_config.dart';
import 'package:vpn_client/core/vpn/vpn_service.dart';
import 'package:vpn_client/presentation/screens/home_screen.dart';
import 'package:dio/dio.dart';

/// Mock ApiClient — returns empty configs without timers.
class MockApiClient extends ApiClient {
  MockApiClient() : super(createMockDio());

  @override
  Future<List<VpnConfig>> getConfigs() async => [];

  @override
  Future<bool> healthCheck() async => true;
}

Dio createMockDio() {
  final dio = Dio();
  dio.httpClientAdapter = NoopHttpClientAdapter();
  return dio;
}

class NoopHttpClientAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    throw UnimplementedError('not used');
  }

  @override
  void close({bool force = false}) {}
}

// Mock VpnService for testing UI in isolation
class MockVpnService implements VpnService {
  final _stateController = StreamController<VpnConnectionState>.broadcast();
  VpnConnectionState _state = VpnConnectionState.disconnected;

  @override
  VpnConnectionState get currentState => _state;

  @override
  Stream<VpnConnectionState> get state => _stateController.stream;

  @override
  Future<void> connect(VpnConfig config) async {
    _setState(VpnConnectionState.connecting);
    await Future.delayed(const Duration(milliseconds: 100));
    _setState(VpnConnectionState.connected);
  }

  @override
  Future<void> disconnect() async {
    _setState(VpnConnectionState.disconnected);
  }

  @override
  void dispose() {
    _stateController.close();
  }

  void _setState(VpnConnectionState newState) {
    _state = newState;
    _stateController.add(newState);
  }
}

void main() {
  group('HomeScreen', () {
    late ApiClient apiClient;
    late MockVpnService vpnService;

    setUp(() {
      apiClient = MockApiClient();
      vpnService = MockVpnService();
    });

    tearDown(() async {
      // Allow pending microtasks from _fetchConfigs to drain
      await Future<void>.delayed(Duration.zero);
    });

    /// Pumps twice to fully initialize HomeScreen (initState → _fetchConfigs → rebuild).
    Future<void> pumpHomeScreen(WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: HomeScreen(
            apiClient: apiClient,
            vpnService: vpnService,
          ),
        ),
      );
      // First pump: process initial build + drain microtasks
      await tester.pump();
      // Second pump: process setState from _fetchConfigs completion
      await tester.pump();
    }

    testWidgets('shows disconnected state initially', (tester) async {
      await pumpHomeScreen(tester);

      expect(find.text('Disconnected'), findsOneWidget);
      expect(find.text('CONNECT'), findsOneWidget);
      expect(find.text('DISCONNECT'), findsNothing);
    });

    testWidgets('shows error state when no server available', (tester) async {
      await pumpHomeScreen(tester);

      // Tap connect button — fails because no configs available
      await tester.tap(find.text('CONNECT'));
      await tester.pump();

      expect(find.textContaining('No configs available'), findsOneWidget);
    });

    testWidgets('debug menu opens on long press', (tester) async {
      await pumpHomeScreen(tester);

      // Long press the shield icon
      await tester.longPress(find.byIcon(Icons.shield_outlined));
      await tester.pump();

      expect(find.text('Debug Settings'), findsOneWidget);
      expect(find.text('SAVE'), findsOneWidget);
    });

    testWidgets('shows app title', (tester) async {
      await pumpHomeScreen(tester);

      expect(find.text('VPN Client'), findsOneWidget);
    });
  });
}
