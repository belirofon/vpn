import 'dart:async';
import 'package:flutter/material.dart';
import '../../data/api/api_client.dart';
import '../../data/models/vpn_config.dart';
import '../../core/vpn/vpn_service.dart';

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
  VpnConfig? _activeConfig;
  StreamSubscription<VpnConnectionState>? _stateSub;
  String? _errorMessage;
  late AnimationController _pulseController;
  late Animation<double> _pulseAnimation;

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
        _activeConfig = null;
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

    try {
      final config = await widget.apiClient.getBestConfig();
      if (!mounted) return;

      if (config == null) {
        setState(() {
          _errorMessage = 'Server unavailable.\nCheck that the server is running.';
        });
        return;
      }

      await widget.vpnService.connect(config);
      if (mounted) setState(() => _activeConfig = config);
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
      builder: (_) => _DebugSheet(apiClient: widget.apiClient),
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
                if (_activeConfig != null)
                  _ServerInfoCard(config: _activeConfig!),
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

// -- Server info card --

class _ServerInfoCard extends StatelessWidget {
  final VpnConfig config;

  const _ServerInfoCard({required this.config});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.dns_outlined, size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text(config.name, style: theme.textTheme.bodyLarge),
            const SizedBox(width: 4),
            Text(
              '(${config.country})',
              style: theme.textTheme.bodySmall?.copyWith(color: Colors.grey),
            ),
            const SizedBox(width: 16),
            Icon(Icons.speed, size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 4),
            Text(
              '${config.latencyMs}ms',
              style: theme.textTheme.bodyLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// -- Debug settings sheet --

class _DebugSheet extends StatefulWidget {
  final ApiClient apiClient;

  const _DebugSheet({required this.apiClient});

  @override
  State<_DebugSheet> createState() => _DebugSheetState();
}

class _DebugSheetState extends State<_DebugSheet> {
  late TextEditingController _urlController;
  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _urlController = TextEditingController(
      text: widget.apiClient.serverUrl ?? '',
    );
  }

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  Future<void> _saveUrl() async {
    final url = _urlController.text.trim();
    if (url.isEmpty) return;

    setState(() => _isSaving = true);
    await widget.apiClient.saveServerUrl(url);

    if (mounted) {
      setState(() => _isSaving = false);
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Server URL saved'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 24,
        right: 24,
        top: 24,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.build_outlined, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text(
                'Debug Settings',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            'Server URL',
            style: theme.textTheme.bodySmall?.copyWith(
              color: Colors.grey,
            ),
          ),
          const SizedBox(height: 6),
          TextField(
            controller: _urlController,
            decoration: InputDecoration(
              hintText: 'http://localhost:8080',
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
              ),
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 12,
              ),
              isDense: true,
            ),
          ),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: _isSaving ? null : _saveUrl,
              child: _isSaving
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('SAVE'),
            ),
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }
}
