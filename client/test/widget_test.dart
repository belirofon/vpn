import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vpn_client/data/api/api_client.dart';
import 'package:vpn_client/data/models/vpn_config.dart';
import 'package:vpn_client/core/vpn/vpn_service.dart';
import 'package:vpn_client/presentation/screens/home_screen.dart';
import 'package:dio/dio.dart';

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
      apiClient = ApiClient(Dio());
      vpnService = MockVpnService();
    });

    testWidgets('shows disconnected state initially', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: HomeScreen(
            apiClient: apiClient,
            vpnService: vpnService,
          ),
        ),
      );

      expect(find.text('Disconnected'), findsOneWidget);
      expect(find.text('CONNECT'), findsOneWidget);
      expect(find.text('DISCONNECT'), findsNothing);
    });

    testWidgets('shows error state when no server available', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: HomeScreen(
            apiClient: apiClient,
            vpnService: vpnService,
          ),
        ),
      );

      // Tap connect button — fails because no server, shows error
      await tester.tap(find.text('CONNECT'));
      await tester.pumpAndSettle();

      expect(find.textContaining('Server unavailable'), findsOneWidget);
    });

    testWidgets('debug menu opens on long press', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: HomeScreen(
            apiClient: apiClient,
            vpnService: vpnService,
          ),
        ),
      );

      // Long press the shield icon
      await tester.longPress(find.byIcon(Icons.shield_outlined));
      await tester.pump();

      expect(find.text('Debug Settings'), findsOneWidget);
      expect(find.text('SAVE'), findsOneWidget);
    });

    testWidgets('shows app title', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: HomeScreen(
            apiClient: apiClient,
            vpnService: vpnService,
          ),
        ),
      );

      expect(find.text('VPN Client'), findsOneWidget);
    });
  });
}
