import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:dio/dio.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'data/api/api_client.dart';
import 'core/vpn/vpn_service.dart';
import 'core/vpn/mobile_vpn_service.dart';
import 'core/vpn/web_vpn_service.dart';
import 'presentation/screens/home_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final prefs = await SharedPreferences.getInstance();
  final dio = Dio();
  final apiClient = ApiClient(dio);
  await apiClient.init(prefs);

  // Read SERVER_URL from --dart-define (build-time override)
  const compileUrl = String.fromEnvironment('SERVER_URL');
  if (compileUrl.isNotEmpty) {
    apiClient.setBaseUrl(compileUrl);
    // Persist so it survives next launch without --dart-define
    await apiClient.saveServerUrl(compileUrl);
  }

  final vpnService = _createVpnService();

  runApp(VpnApp(apiClient: apiClient, vpnService: vpnService));
}

VpnService _createVpnService() {
  if (kIsWeb) {
    return WebVpnService();
  }
  return MobileVpnService();
}

class VpnApp extends StatelessWidget {
  final ApiClient apiClient;
  final VpnService vpnService;

  const VpnApp({
    super.key,
    required this.apiClient,
    required this.vpnService,
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
      home: HomeScreen(apiClient: apiClient, vpnService: vpnService),
    );
  }
}
