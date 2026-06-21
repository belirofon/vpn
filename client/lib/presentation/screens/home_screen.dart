import 'package:flutter/material.dart';
import '../../data/api/api_client.dart';
import '../../domain/entities/vpn_config.dart';
import '../viewmodels/home_viewmodel.dart';
import 'admin_login_screen.dart';

class HomeScreen extends StatefulWidget {
  final ApiClient apiClient;
  final HomeViewModel viewModel;

  const HomeScreen({
    super.key,
    required this.apiClient,
    required this.viewModel,
  });

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen>
    with SingleTickerProviderStateMixin {
  late AnimationController _pulseController;
  late Animation<double> _pulseAnimation;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
    _pulseAnimation = Tween<double>(begin: 0.85, end: 1.0).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );
    widget.viewModel.addListener(_onViewModelChanged);
  }

  @override
  void dispose() {
    widget.viewModel.removeListener(_onViewModelChanged);
    _pulseController.dispose();
    super.dispose();
  }

  void _onViewModelChanged() {
    if (widget.viewModel.isLoading) {
      _pulseController.repeat(reverse: true);
    } else {
      _pulseController.stop();
      _pulseController.value = 1.0;
    }
  }

  void _showDebugMenu() {
    showModalBottomSheet(
      context: context,
      builder: (_) => _DebugSheet(apiClient: widget.apiClient),
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
    return ListenableBuilder(
      listenable: widget.viewModel,
      builder: (context, _) {
        final vm = widget.viewModel;
        final theme = Theme.of(context);
        final statusColor = switch (vm.screenState) {
          ScreenState.connected => Colors.green,
          ScreenState.error => theme.colorScheme.error,
          _ => Colors.grey,
        };

        return Scaffold(
          appBar: AppBar(
            title: const Text('VPN Client'),
            centerTitle: true,
            actions: [
              Padding(
                padding: const EdgeInsets.only(right: 4),
                child:               IconButton(
                  icon: const Icon(Icons.admin_panel_settings_outlined),
                  tooltip: 'Admin Panel',
                  onPressed: _openAdminLogin,
                ),
              ),
            ],
          ),
          body: SafeArea(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Column(
                children: [
                  const SizedBox(height: 28),
                  _ModeSwitch(
                    selectedMode: vm.selectedMode,
                    onChanged: vm.selectMode,
                  ),
                  const Spacer(),
                  GestureDetector(
                    onLongPress: _showDebugMenu,
                    child: _ShieldWidget(
                      isLoading: vm.isLoading,
                      isConnected: vm.isConnected,
                      statusColor: statusColor,
                      pulseAnimation: _pulseAnimation,
                    ),
                  ),
                  const SizedBox(height: 24),
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
                        vm.statusLabel,
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w600,
                          color: statusColor,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),
                  if (vm.activeConfig != null)
                    _ServerInfoCard(config: vm.activeConfig!),
                  if (vm.errorMessage != null) ...[
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
                              vm.errorMessage!,
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
                  const Spacer(),
                  Padding(
                    padding: const EdgeInsets.only(bottom: 24),
                    child: SizedBox(
                      width: double.infinity,
                      height: 56,
                      child: vm.isConnected
                          ? _DisconnectButton(onPressed: vm.toggleConnection)
                          : _ConnectButton(
                              isLoading: vm.isLoading,
                              onPressed: vm.toggleConnection,
                            ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

// -- Mode Switch --

class _ModeSwitch extends StatelessWidget {
  final HomeMode selectedMode;
  final ValueChanged<HomeMode> onChanged;

  const _ModeSwitch({
    required this.selectedMode,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: MediaQuery.of(context).size.width * 0.8,
      height: 48,
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Expanded(child: _ModeOption(
            label: 'WARP',
            icon: Icons.bolt,
            isSelected: selectedMode == HomeMode.warp,
            onTap: () => onChanged(HomeMode.warp),
          )),
          Expanded(child: _ModeOption(
            label: 'Proxy',
            icon: Icons.shield_outlined,
            isSelected: selectedMode == HomeMode.proxy,
            onTap: () => onChanged(HomeMode.proxy),
          )),
        ],
      ),
    );
  }
}

class _ModeOption extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool isSelected;
  final VoidCallback onTap;

  const _ModeOption({
    required this.label,
    required this.icon,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        height: 48,
        decoration: BoxDecoration(
          color: isSelected ? const Color(0xFFE8F5E9) : Colors.transparent,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(
            color: isSelected ? const Color(0xFF4CAF50) : Colors.transparent,
            width: 1.5,
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 20,
                color: isSelected ? const Color(0xFF2E7D32) : const Color(0xFF616161)),
            const SizedBox(width: 6),
            Text(label, style: TextStyle(
              fontWeight: FontWeight.w600,
              color: isSelected ? const Color(0xFF2E7D32) : const Color(0xFF616161),
            )),
          ],
        ),
      ),
    );
  }
}

// -- Shield Widget --

class _ShieldWidget extends StatelessWidget {
  final bool isLoading;
  final bool isConnected;
  final Color statusColor;
  final Animation<double> pulseAnimation;

  const _ShieldWidget({
    required this.isLoading,
    required this.isConnected,
    required this.statusColor,
    required this.pulseAnimation,
  });

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: pulseAnimation,
      builder: (context, child) {
        return Transform.scale(
          scale: isLoading ? pulseAnimation.value : 1.0,
          child: child,
        );
      },
      child: Stack(
        alignment: Alignment.center,
        children: [
          if (isConnected) ...[
            Container(
              width: 140,
              height: 140,
              decoration: const BoxDecoration(
                shape: BoxShape.circle,
                color: Color(0x1A4CAF50),
              ),
            ),
            Container(
              width: 120,
              height: 120,
              decoration: const BoxDecoration(
                shape: BoxShape.circle,
                color: Color(0x334CAF50),
              ),
            ),
          ],
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 300),
            child: Icon(
              isConnected ? Icons.shield : Icons.shield_outlined,
              key: ValueKey(isConnected),
              size: 96,
              color: statusColor,
            ),
          ),
        ],
      ),
    );
  }
}

// -- Server Info Card --

class _ServerInfoCard extends StatelessWidget {
  final VpnConfig config;

  const _ServerInfoCard({required this.config});

  @override
  Widget build(BuildContext context) {
    final latencyColor = config.latencyMs < 60
        ? Colors.green
        : config.latencyMs < 120
            ? Colors.orange
            : Colors.red;

    return Card(
      elevation: 0,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        child: Row(
          children: [
            Icon(Icons.dns_outlined, size: 18, color: Colors.grey.shade600),
            const SizedBox(width: 8),
            Text(
              config.name,
              style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
            ),
            const SizedBox(width: 4),
            Text(
              '(${config.country})',
              style: TextStyle(fontSize: 12, color: Colors.grey.shade500),
            ),
            const Spacer(),
            Icon(Icons.speed, size: 18, color: latencyColor),
            const SizedBox(width: 4),
            Text(
              '${config.latencyMs}ms',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: latencyColor,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// -- Connect Button --

class _ConnectButton extends StatelessWidget {
  final bool isLoading;
  final VoidCallback onPressed;

  const _ConnectButton({
    required this.isLoading,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return FilledButton(
      onPressed: isLoading ? null : onPressed,
      style: FilledButton.styleFrom(
        backgroundColor: theme.colorScheme.primary,
        disabledBackgroundColor: Colors.grey.shade300,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      child: isLoading
          ? Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(
                  width: 22,
                  height: 22,
                  child: CircularProgressIndicator(
                    strokeWidth: 2.5,
                    color: Colors.white,
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
          : const Text(
              'CONNECT',
              style: TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.bold,
              ),
            ),
    );
  }
}

// -- Disconnect Button --

class _DisconnectButton extends StatelessWidget {
  final VoidCallback onPressed;

  const _DisconnectButton({required this.onPressed});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFFE53935), Color(0xFFC62828)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onPressed,
          borderRadius: BorderRadius.circular(16),
          child: Container(
            height: 56,
            alignment: Alignment.center,
            child: const Text(
              'DISCONNECT',
              style: TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.w700,
                color: Colors.white,
                letterSpacing: 1.2,
              ),
            ),
          ),
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
              hintText: 'https://belirofon-vpn.duckdns.org:8443',
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
