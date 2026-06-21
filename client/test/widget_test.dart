import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vpn_client/data/api/api_client.dart';
import 'package:vpn_client/domain/entities/vpn_config.dart';
import 'package:vpn_client/domain/entities/warp_config.dart';
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

class MockApiClient extends ApiClient {
  @override
  Future<List<VpnConfig>> getConfigs() async => [];

  @override
  Future<WarpConfig?> getWarpConfig() async => null;
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
    late MockApiClient apiClient;
    late MockVpnService vpnService;
    late HomeViewModel viewModel;

    setUp(() {
      apiClient = MockApiClient();
      vpnService = MockVpnService();
      viewModel = HomeViewModel(
        apiClient: apiClient,
        vpnService: vpnService,
      );
    });

    testWidgets('shows disconnected state initially', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));
      await tester.pump(); // settle loadConfigs

      // Disconnected state shows the mode switch and config list
      expect(find.text('WARP'), findsOneWidget);
      expect(find.text('Proxy'), findsOneWidget);
      expect(find.text('No proxy configs available'), findsOneWidget);
      expect(find.text('DISCONNECT'), findsNothing);
    });

    testWidgets('shows error state when no server available', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));
      await tester.pump(); // settle loadConfigs

      expect(find.text('No proxy configs available'), findsOneWidget);
    });

    testWidgets('shows proxy config list on startup', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));
      await tester.pump();

      // Proxy tab is selected by default
      expect(find.text('Proxy'), findsOneWidget);
      expect(find.text('WARP'), findsOneWidget);
      // Empty list message from mock
      expect(find.text('No proxy configs available'), findsOneWidget);
    });

    testWidgets('shows app title', (tester) async {
      await tester.pumpWidget(_buildTestApp(apiClient, viewModel));
      await tester.pump(); // settle loadConfigs

      expect(find.text('VPN Client'), findsOneWidget);
    });
  });
}
