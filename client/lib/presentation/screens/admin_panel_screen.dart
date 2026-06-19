import 'package:flutter/material.dart';
import '../../data/api/api_client.dart';

class AdminPanelScreen extends StatefulWidget {
  final ApiClient apiClient;

  const AdminPanelScreen({super.key, required this.apiClient});

  @override
  State<AdminPanelScreen> createState() => _AdminPanelScreenState();
}

class _AdminPanelScreenState extends State<AdminPanelScreen> {
  bool _isLoading = true;
  String _error = '';
  Map<String, dynamic>? _health;
  Map<String, dynamic>? _endpoints;
  Map<String, dynamic>? _config;

  // Subscription URL editing
  final _subscriptionController = TextEditingController();
  final _refreshIntervalController = TextEditingController();
  bool _isSavingConfig = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  @override
  void dispose() {
    _subscriptionController.dispose();
    _refreshIntervalController.dispose();
    super.dispose();
  }

  Future<void> _loadData() async {
    setState(() {
      _isLoading = true;
      _error = '';
    });

    final results = await Future.wait([
      widget.apiClient.adminHealth(),
      widget.apiClient.adminEndpoints(),
      widget.apiClient.adminGetConfig(),
    ]);

    if (!mounted) return;

    setState(() {
      _isLoading = false;
      _health = results[0];
      _endpoints = results[1];
      _config = results[2];
      if (results[0] == null && results[2] == null) {
        _error = 'Failed to load admin data.\nCheck server connectivity.';
      }
      _subscriptionController.text =
          _config?['subscription_url']?.toString() ?? '';
      _refreshIntervalController.text =
          _config?['refresh_interval']?.toString() ?? '30m0s';
    });
  }

  Future<void> _saveConfig() async {
    setState(() => _isSavingConfig = true);

    final ok = await widget.apiClient.adminUpdateConfig(
      subscriptionUrl: _subscriptionController.text.trim(),
      refreshInterval: _refreshIntervalController.text.trim(),
    );

    if (!mounted) return;
    setState(() => _isSavingConfig = false);

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? 'Configuration updated' : 'Failed to update config'),
        behavior: SnackBarBehavior.floating,
        backgroundColor: ok ? Colors.green : Theme.of(context).colorScheme.error,
      ),
    );
    if (ok) _loadData();
  }

  Future<void> _refreshConfigs() async {
    final ok = await widget.apiClient.adminRefreshConfigs();

    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? 'Config refresh triggered' : 'Failed to trigger refresh'),
        behavior: SnackBarBehavior.floating,
      ),
    );
    if (ok) {
      await Future.delayed(const Duration(seconds: 2));
      if (mounted) _loadData();
    }
  }

  Future<void> _logout() async {
    await widget.apiClient.adminLogout();
    if (!mounted) return;
    Navigator.popUntil(context, (route) => route.isFirst);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Admin Panel'),
        centerTitle: true,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _loadData,
          ),
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'Logout',
            onPressed: _logout,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty && _health == null
              ? Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        const Icon(Icons.cloud_off, size: 64, color: Colors.grey),
                        const SizedBox(height: 16),
                        Text(
                          _error,
                          textAlign: TextAlign.center,
                          style: theme.textTheme.bodyLarge,
                        ),
                        const SizedBox(height: 16),
                        FilledButton.tonal(
                          onPressed: _loadData,
                          child: const Text('Retry'),
                        ),
                      ],
                    ),
                  ),
                )
              : RefreshIndicator(
                  onRefresh: _loadData,
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      // -- Health Card --
                      _buildSectionCard(
                        theme,
                        icon: Icons.monitor_heart_outlined,
                        title: 'Server Health',
                        children: [
                          _health != null
                              ? Column(
                                  children: [
                                    _infoRow(theme, 'Status',
                                        _health!['status']?.toString() ?? '-'),
                                    _infoRow(theme, 'Configs Tested',
                                        '${_health!['configs_tested'] ?? '-'}'),
                                    _infoRow(theme, 'Uptime',
                                        _health!['uptime']?.toString() ?? '-'),
                                    _infoRow(theme, 'Subscription',
                                        _health!['subscription_url']?.toString() ?? '-'),
                                    const SizedBox(height: 8),
                                    SizedBox(
                                      width: double.infinity,
                                      child: FilledButton.tonal(
                                        onPressed: _refreshConfigs,
                                        child: const Text('Refresh Configs Now'),
                                      ),
                                    ),
                                  ],
                                )
                              : const Text('No health data',
                                  style: TextStyle(color: Colors.grey)),
                        ],
                      ),
                      const SizedBox(height: 12),

                      // -- Config Card --
                      _buildSectionCard(
                        theme,
                        icon: Icons.settings_outlined,
                        title: 'Configuration',
                        children: [
                          TextField(
                            controller: _subscriptionController,
                            decoration: InputDecoration(
                              labelText: 'SUBSCRIPTION_URL',
                              border: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                              isDense: true,
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 12, vertical: 10),
                            ),
                            style: const TextStyle(fontSize: 13),
                          ),
                          const SizedBox(height: 12),
                          TextField(
                            controller: _refreshIntervalController,
                            decoration: InputDecoration(
                              labelText: 'REFRESH_INTERVAL (e.g. 30m, 1h)',
                              border: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                              isDense: true,
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 12, vertical: 10),
                            ),
                            style: const TextStyle(fontSize: 13),
                          ),
                          const SizedBox(height: 12),
                          SizedBox(
                            width: double.infinity,
                            child: FilledButton(
                              onPressed: _isSavingConfig ? null : _saveConfig,
                              child: _isSavingConfig
                                  ? const SizedBox(
                                      width: 20,
                                      height: 20,
                                      child: CircularProgressIndicator(
                                          strokeWidth: 2),
                                    )
                                  : const Text('Save Config'),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),

                      // -- Endpoints Card --
                      _buildSectionCard(
                        theme,
                        icon: Icons.api_outlined,
                        title: 'API Endpoints',
                        trailing: _endpoints != null
                            ? Text(
                                '${_endpoints!['total'] ?? 0} endpoints',
                                style: theme.textTheme.bodySmall
                                    ?.copyWith(color: Colors.grey),
                              )
                            : null,
                        children: [
                          if (_endpoints?['endpoints'] is List)
                            ...(_endpoints!['endpoints'] as List).map(
                              (ep) => Padding(
                                padding: const EdgeInsets.symmetric(vertical: 2),
                                child: Row(
                                  children: [
                                    Container(
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 6,
                                        vertical: 2,
                                      ),
                                      decoration: BoxDecoration(
                                        color: _methodColor(
                                                ep['method']?.toString() ?? '')
                                            .withValues(alpha: 0.15),
                                        borderRadius: BorderRadius.circular(4),
                                      ),
                                      child: Text(
                                        ep['method']?.toString() ?? '',
                                        style: TextStyle(
                                          fontSize: 11,
                                          fontWeight: FontWeight.bold,
                                          color: _methodColor(
                                              ep['method']?.toString() ?? ''),
                                        ),
                                      ),
                                    ),
                                    const SizedBox(width: 12),
                                    Expanded(
                                      child: Text(
                                        ep['path']?.toString() ?? '',
                                        style: const TextStyle(
                                          fontSize: 13,
                                          fontFamily: 'monospace',
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            )
                          else
                            const Text('No endpoint data',
                                style: TextStyle(color: Colors.grey)),
                        ],
                      ),
                      const SizedBox(height: 12),

                      // -- Server Info Card (read-only config values) --
                      if (_config != null)
                        _buildSectionCard(
                          theme,
                          icon: Icons.info_outline,
                          title: 'Server Settings',
                          children: [
                            _infoRow(theme, 'Ping Timeout',
                                _config!['ping_timeout']?.toString() ?? '-'),
                            _infoRow(theme, 'Mock Configs',
                                '${_config!['mock_configs'] ?? '-'}'),
                            _infoRow(theme, 'Skip Verify TLS',
                                '${_config!['skip_verify_tls'] ?? '-'}'),
                            _infoRow(theme, 'CORS Origins',
                                _config!['cors_origins']?.toString() ?? '-'),
                          ],
                        ),
                      const SizedBox(height: 24),
                      // Bottom padding
                    ],
                  ),
                ),
    );
  }

  Widget _buildSectionCard(
    ThemeData theme, {
    required IconData icon,
    required String title,
    Widget? trailing,
    required List<Widget> children,
  }) {
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 20, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  title,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                if (trailing != null) trailing,
              ],
            ),
            const SizedBox(height: 12),
            ...children,
          ],
        ),
      ),
    );
  }

  Widget _infoRow(ThemeData theme, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Text(
            '$label: ',
            style: theme.textTheme.bodySmall?.copyWith(
              color: Colors.grey,
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w500,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }

  Color _methodColor(String method) {
    return switch (method) {
      'GET' => Colors.green,
      'POST' => Colors.orange,
      'PUT' => Colors.blue,
      'DELETE' => Colors.red,
      'PATCH' => Colors.purple,
      _ => Colors.grey,
    };
  }
}
