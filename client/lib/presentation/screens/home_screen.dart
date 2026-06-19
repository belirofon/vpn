import 'dart:async';
import 'package:flutter/material.dart';
import '../../data/api/api_client.dart';
import '../../data/models/vpn_config.dart';
import '../../core/vpn/vpn_service.dart';
import '../widgets/server_info_card.dart';
import '../widgets/debug_sheet.dart';
import 'admin_login_screen.dart';

class HomeScreen extends StatefulWidget {
  final ApiClient apiClient;
  final VpnService vpnService;

  const HomeScreen({
    super.key,
    required this.apiClient,
    required this.vpnService,
  });

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen>
    with SingleTickerProviderStateMixin {
  VpnConnectionState _connectionState = VpnConnectionState.disconnected;
  StreamSubscription<VpnConnectionState>? _stateSub;
  String? _errorMessage;
  late AnimationController _pulseController;
  late Animation<double> _pulseAnimation;

  List<VpnConfig> _configs = [];
  int _currentIndex = 0;
  bool _isLoadingConfigs = true;

  @override
  void initState() {
    super.initState();
    _stateSub = widget.vpnService.state.listen(_onStateChanged);

    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
    _pulseAnimation = Tween<double>(begin: 0.85, end: 1.0).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );

    _fetchConfigs();
  }

  Future<void> _fetchConfigs() async {
    setState(() => _isLoadingConfigs = true);
    final configs = await widget.apiClient.getConfigs();
    if (!mounted) return;
    setState(() {
      _configs = configs.take(10).toList();
      _currentIndex = 0;
      _isLoadingConfigs = false;
    });
  }

  void _goToPrev() {
    if (_currentIndex > 0) {
      setState(() => _currentIndex--);
    }
  }

  void _goToNext() {
    if (_currentIndex < _configs.length - 1) {
      setState(() => _currentIndex++);
    }
  }

  @override
  void dispose() {
    _stateSub?.cancel();
    _pulseController.dispose();
    super.dispose();
  }

  void _onStateChanged(VpnConnectionState state) {
    if (!mounted) return;
    setState(() {
      _connectionState = state;
      if (state == VpnConnectionState.disconnected) {
        _errorMessage = null;
      }
    });

    if (state == VpnConnectionState.connecting) {
      _pulseController.repeat(reverse: true);
    } else {
      _pulseController.stop();
      _pulseController.value = 1.0;
    }
  }

  Future<void> _toggleConnection() async {
    if (_connectionState == VpnConnectionState.connected) {
      await widget.vpnService.disconnect();
      return;
    }

    setState(() => _errorMessage = null);

    if (_configs.isEmpty) {
      setState(() {
        _errorMessage = 'No configs available.\nCheck that the server is running.';
      });
      return;
    }

    try {
      final config = _configs[_currentIndex];
      await widget.vpnService.connect(config);
    } catch (e) {
      if (mounted) {
        _showErrorSnackBar(e.toString());
      }
    }
  }

  void _showErrorSnackBar(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        backgroundColor: Theme.of(context).colorScheme.error,
      ),
    );
  }

  void _showDebugMenu() {
    showModalBottomSheet(
      context: context,
      builder: (_) => DebugSheet(apiClient: widget.apiClient),
    );
  }

  void _openAdminLogin() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => AdminLoginScreen(apiClient: widget.apiClient),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isConnected = _connectionState == VpnConnectionState.connected;
    final isLoading = _connectionState == VpnConnectionState.connecting;
    final statusColor = switch (_connectionState) {
      VpnConnectionState.connected => Colors.green,
      VpnConnectionState.error => theme.colorScheme.error,
      _ => Colors.grey,
    };

    return Scaffold(
      appBar: AppBar(
        title: const Text('VPN Client'),
        centerTitle: true,
        actions: [
          IconButton(
            icon: const Icon(Icons.admin_panel_settings_outlined),
            tooltip: 'Admin',
            onPressed: _openAdminLogin,
          ),
        ],
      ),
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Spacer(flex: 2),
                // Animated shield
                GestureDetector(
                  onLongPress: _showDebugMenu,
                  child: AnimatedBuilder(
                    animation: _pulseAnimation,
                    builder: (context, child) {
                      return Transform.scale(
                        scale: isLoading ? _pulseAnimation.value : 1.0,
                        child: child,
                      );
                    },
                    child: AnimatedSwitcher(
                      duration: const Duration(milliseconds: 300),
                      child: Icon(
                        isConnected
                            ? Icons.shield
                            : Icons.shield_outlined,
                        key: ValueKey(isConnected),
                        size: 120,
                        color: statusColor,
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 24),
                // Status row
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Container(
                      width: 10,
                      height: 10,
                      decoration: BoxDecoration(
                        color: statusColor,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      _statusLabel(),
                      style: theme.textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: statusColor,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                // Server info
                if (_isLoadingConfigs)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 16),
                    child: SizedBox(
                      width: 24, height: 24,
                      child: CircularProgressIndicator(strokeWidth: 2.5),
                    ),
                  )
                else if (_configs.isNotEmpty)
                  ServerInfoCard(
                    config: _configs[_currentIndex],
                    currentIndex: _currentIndex,
                    totalCount: _configs.length,
                    onPrev: _goToPrev,
                    onNext: _goToNext,
                  ),
                // Error message
                if (_errorMessage != null) ...[
                  const SizedBox(height: 16),
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.errorContainer,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Row(
                      children: [
                        Icon(Icons.warning_rounded,
                            color: theme.colorScheme.onErrorContainer,
                            size: 20),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            _errorMessage!,
                            style: TextStyle(
                              color: theme.colorScheme.onErrorContainer,
                              fontSize: 13,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
                const Spacer(flex: 2),
                // Action button
                SizedBox(
                  width: double.infinity,
                  height: 56,
                  child: FilledButton(
                    onPressed: isLoading ? null : _toggleConnection,
                    style: FilledButton.styleFrom(
                      backgroundColor: isConnected
                          ? theme.colorScheme.error
                          : theme.colorScheme.primary,
                      disabledBackgroundColor: Colors.grey.shade300,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(16),
                      ),
                    ),
                    child: isLoading
                        ? Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              SizedBox(
                                width: 22,
                                height: 22,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2.5,
                                  color: theme.colorScheme.onPrimary,
                                ),
                              ),
                              const SizedBox(width: 12),
                              Text(
                                'CONNECTING…',
                                style: TextStyle(
                                  fontSize: 16,
                                  fontWeight: FontWeight.bold,
                                  color: theme.colorScheme.onPrimary,
                                ),
                              ),
                            ],
                          )
                        : Text(
                            isConnected ? 'DISCONNECT' : 'CONNECT',
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                  ),
                ),
                const Spacer(flex: 1),
              ],
            ),
          ),
        ),
      ),
    );
  }

  String _statusLabel() {
    return switch (_connectionState) {
      VpnConnectionState.connected => 'Connected',
      VpnConnectionState.connecting => 'Connecting…',
      VpnConnectionState.error => 'Error',
      VpnConnectionState.disconnected => 'Disconnected',
    };
  }
}
