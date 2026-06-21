import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vpn_client/data/api/api_client.dart';
import 'package:vpn_client/domain/entities/vpn_config.dart';
import 'package:vpn_client/domain/services/vpn_service.dart';
import 'package:vpn_client/presentation/screens/home_screen.dart';
import 'package:vpn_client/presentation/viewmodels/home_viewmodel.dart';

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

Widget _buildTestApp(ApiClient apiClient, HomeViewModel viewModel) {
  return MaterialApp(
    home: HomeScreen(
      apiClient: apiClient,
      viewModel: viewModel,
    ),
  );
}

void main() {
  group('HomeScreen', () {
    late ApiClient apiClient;
    late MockVpnService vpnService;
    late HomeViewModel viewModel;

    setUp(() {
      apiClient = ApiClient();
      vpnService = MockVpnService();
      viewModel = HomeViewModel(
        apiClient: apiClient,
        vpnService: vpnService,
      );
    });

    testWidgets('shows disconnected state initially', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));

      expect(find.text('Disconnected'), findsOneWidget);
      expect(find.text('CONNECT'), findsOneWidget);
      expect(find.text('DISCONNECT'), findsNothing);
    });

    testWidgets('shows error state when no server available', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));

      await tester.tap(find.text('CONNECT'));
      await tester.pumpAndSettle();

      expect(find.textContaining('Server unavailable'), findsOneWidget);
    });

    testWidgets('debug menu opens on long press', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));

      await tester.longPress(find.byIcon(Icons.shield_outlined).last);
      await tester.pump();

      expect(find.text('Debug Settings'), findsOneWidget);
      expect(find.text('SAVE'), findsOneWidget);
    });

    testWidgets('shows app title', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));

      expect(find.text('VPN Client'), findsOneWidget);
    });
  });
}
