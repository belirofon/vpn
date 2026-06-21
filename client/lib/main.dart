import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'core/vpn/mobile_vpn_service.dart';
import 'core/vpn/web_vpn_service.dart';
import 'data/api/api_client.dart';
import 'domain/services/vpn_service.dart';
import 'presentation/screens/home_screen.dart';
import 'presentation/viewmodels/home_viewmodel.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final prefs = await SharedPreferences.getInstance();
  final apiClient = ApiClient();
  await apiClient.init(prefs);

  const compileUrl = String.fromEnvironment('SERVER_URL');
  if (compileUrl.isNotEmpty) {
    apiClient.setBaseUrl(compileUrl);
    await apiClient.saveServerUrl(compileUrl);
  }

  final vpnService = _createVpnService();
  final viewModel = HomeViewModel(
    apiClient: apiClient,
    vpnService: vpnService,
  );

  runApp(VpnApp(
    apiClient: apiClient,
    viewModel: viewModel,
  ));
}

VpnService _createVpnService() {
  if (kIsWeb) {
    return WebVpnService();
  }
  return MobileVpnService();
}

class VpnApp extends StatelessWidget {
  final ApiClient apiClient;
  final HomeViewModel viewModel;

  const VpnApp({
    super.key,
    required this.apiClient,
    required this.viewModel,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'VPN Client',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
        useMaterial3: true,
      ),
      home: HomeScreen(
        apiClient: apiClient,
        viewModel: viewModel,
      ),
    );
  }
}
