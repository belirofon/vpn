import 'package:flutter/material.dart';
import '../../data/api/api_client.dart';
import '../../data/dto/admin_models.dart';
import '../../domain/entities/warp_config.dart';
import '../viewmodels/admin_viewmodel.dart';

class AdminPanelScreen extends StatefulWidget {
  final ApiClient apiClient;

  const AdminPanelScreen({super.key, required this.apiClient});

  @override
  State<AdminPanelScreen> createState() => _AdminPanelScreenState();
}

class _AdminPanelScreenState extends State<AdminPanelScreen> {
  late AdminViewModel _viewModel;
  late TextEditingController _subscriptionController;
  late TextEditingController _refreshIntervalController;
  bool _controllersInitialized = false;

  @override
  void initState() {
    super.initState();
    _viewModel = AdminViewModel(apiClient: widget.apiClient);
    _subscriptionController = TextEditingController();
    _refreshIntervalController = TextEditingController();
    _viewModel.addListener(_onDataLoaded);
    _viewModel.loadData();
    _viewModel.loadWarp();
  }

  @override
  void dispose() {
    _viewModel.removeListener(_onDataLoaded);
    _subscriptionController.dispose();
    _refreshIntervalController.dispose();
    super.dispose();
  }

  void _onDataLoaded() {
    if (_viewModel.config != null && !_controllersInitialized) {
      _controllersInitialized = true;
      _subscriptionController.text =
          _viewModel.config!.subscriptionUrl;
      _refreshIntervalController.text =
          _viewModel.config!.refreshInterval;
    }
  }

  Future<void> _saveConfig() async {
    final ok = await _viewModel.saveConfig(
      subscriptionUrl: _subscriptionController.text.trim(),
      refreshInterval: _refreshIntervalController.text.trim(),
    );

    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
            ok ? 'Configuration updated' : 'Failed to update config'),
        behavior: SnackBarBehavior.floating,
        backgroundColor:
            ok ? Colors.green : Theme.of(context).colorScheme.error,
      ),
    );
  }

  Future<void> _refreshConfigs() async {
    final ok = await _viewModel.refreshConfigs();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok
            ? 'Config refresh triggered'
            : 'Failed to trigger refresh'),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  Future<void> _generateWarp() async {
    final config = await _viewModel.generateWarp();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(config != null
            ? 'WARP config generated (${config.latencyMs}ms)'
            : 'Failed to generate WARP config'),
        behavior: SnackBarBehavior.floating,
        backgroundColor: config != null
            ? Colors.green
            : Theme.of(context).colorScheme.error,
      ),
    );
  }

  Future<void> _deleteWarp() async {
    final ok = await _viewModel.deleteWarp();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content:
            Text(ok ? 'WARP config deleted' : 'Failed to delete WARP config'),
        behavior: SnackBarBehavior.floating,
        backgroundColor:
            ok ? Colors.green : Theme.of(context).colorScheme.error,
      ),
    );
  }

  Future<void> _logout() async {
    await _viewModel.logout();
    if (!mounted) return;
    Navigator.popUntil(context, (route) => route.isFirst);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Admin Panel'),
        centerTitle: true,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _viewModel.loadData,
          ),
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'Logout',
            onPressed: _logout,
          ),
        ],
      ),
      body: ListenableBuilder(
        listenable: _viewModel,
        builder: (context, _) {
          if (_viewModel.isLoading) {
            return const Center(child: CircularProgressIndicator());
          }

          if (_viewModel.error.isNotEmpty && _viewModel.health == null) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.cloud_off,
                        size: 64, color: Colors.grey),
                    const SizedBox(height: 16),
                    Text(
                      _viewModel.error,
                      textAlign: TextAlign.center,
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 16),
                    FilledButton.tonal(
                      onPressed: _viewModel.loadData,
                      child: const Text('Retry'),
                    ),
                  ],
                ),
              ),
            );
          }

          return RefreshIndicator(
            onRefresh: _viewModel.loadData,
            child: ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: _sectionCount(),
              itemBuilder: _buildSection,
            ),
          );
        },
      ),
    );
  }

  int _sectionCount() {
    int count = 4; // health, config, endpoints, warp
    if (_viewModel.config != null) count++;
    return count;
  }

  Widget _buildSection(BuildContext context, int index) {
    switch (index) {
      case 0:
        return _HealthSection(
          health: _viewModel.health,
          onRefreshConfigs: _refreshConfigs,
        );
      case 1:
        return _ConfigSection(
          subscriptionController: _subscriptionController,
          refreshIntervalController: _refreshIntervalController,
          isSaving: _viewModel.isSavingConfig,
          onSave: _saveConfig,
        );
      case 2:
        return _EndpointsSection(
          endpoints: _viewModel.endpoints,
        );
      case 3:
        return _WarpSection(
          warpStatus: _viewModel.warpStatus,
          isLoading: _viewModel.isWarpLoading,
          isGenerating: _viewModel.isWarpGenerating,
          onGenerate: _generateWarp,
          onDelete: _deleteWarp,
        );
      case 4:
        if (_viewModel.config != null) {
          return _ServerSettingsSection(config: _viewModel.config!);
        }
        return const SizedBox.shrink();
      default:
        return const SizedBox.shrink();
    }
  }
}

// -- Health Section --

class _HealthSection extends StatelessWidget {
  final AdminHealth? health;
  final VoidCallback onRefreshConfigs;

  const _HealthSection({
    required this.health,
    required this.onRefreshConfigs,
  });

  @override
  Widget build(BuildContext context) {
    final health = this.health;

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: _SectionCard(
        icon: Icons.monitor_heart_outlined,
        title: 'Server Health',
        children: [
          if (health != null) ...[
            _InfoRow(label: 'Status', value: health.status),
            _InfoRow(
                label: 'Configs Tested', value: '${health.configsTested}'),
            _InfoRow(label: 'Uptime', value: health.uptime),
            _InfoRow(
                label: 'Subscription', value: health.subscriptionUrl),
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: FilledButton.tonal(
                onPressed: onRefreshConfigs,
                child: const Text('Refresh Configs Now'),
              ),
            ),
          ] else
            const Text('No health data',
                style: TextStyle(color: Colors.grey)),
        ],
      ),
    );
  }
}

// -- Config Section --

class _ConfigSection extends StatelessWidget {
  final TextEditingController subscriptionController;
  final TextEditingController refreshIntervalController;
  final bool isSaving;
  final VoidCallback onSave;

  const _ConfigSection({
    required this.subscriptionController,
    required this.refreshIntervalController,
    required this.isSaving,
    required this.onSave,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: _SectionCard(
        icon: Icons.settings_outlined,
        title: 'Configuration',
        children: [
          TextField(
            controller: subscriptionController,
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
            controller: refreshIntervalController,
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
              onPressed: isSaving ? null : onSave,
              child: isSaving
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child:
                          CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Save Config'),
            ),
          ),
        ],
      ),
    );
  }
}

// -- Endpoints Section --

class _EndpointsSection extends StatelessWidget {
  final List<AdminEndpoint>? endpoints;

  const _EndpointsSection({required this.endpoints});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: _SectionCard(
        icon: Icons.api_outlined,
        title: 'API Endpoints',
        trailing: endpoints != null
            ? Text(
                '${endpoints!.length} endpoints',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: Colors.grey),
              )
            : null,
        children: [
          if (endpoints != null && endpoints!.isNotEmpty)
            ...endpoints!.map(
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
                        color: Color(ep.methodColorValue())
                            .withValues(alpha: 0.15),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        ep.method,
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.bold,
                          color: Color(ep.methodColorValue()),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        ep.path,
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
    );
  }
}

// -- Server Settings Section --

class _ServerSettingsSection extends StatelessWidget {
  final AdminConfig config;

  const _ServerSettingsSection({required this.config});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: _SectionCard(
        icon: Icons.info_outline,
        title: 'Server Settings',
        children: [
          _InfoRow(label: 'Ping Timeout', value: config.pingTimeout),
          _InfoRow(label: 'Mock Configs',
              value: config.mockConfigs.toString()),
          _InfoRow(label: 'Skip Verify TLS',
              value: config.skipVerifyTls.toString()),
          _InfoRow(
              label: 'CORS Origins', value: config.corsOrigins),
        ],
      ),
    );
  }
}

// -- WARP Section --

class _WarpSection extends StatelessWidget {
  final AdminWarpStatus? warpStatus;
  final bool isLoading;
  final bool isGenerating;
  final VoidCallback onGenerate;
  final VoidCallback onDelete;

  const _WarpSection({
    required this.warpStatus,
    required this.isLoading,
    required this.isGenerating,
    required this.onGenerate,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final status = warpStatus;

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: _SectionCard(
        icon: Icons.cloud_outlined,
        title: 'WARP Config',
        trailing: isLoading || isGenerating
            ? const SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : null,
        children: [
          if (isLoading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: Text('Loading...',
                  style: TextStyle(color: Colors.grey)),
            )
          else if (isGenerating)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: Text('Generating WARP config...',
                  style: TextStyle(color: Colors.grey)),
            )
          else if (status != null && status.available && status.config != null)
            _WarpDetails(config: status.config!)
          else
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: Text('WARP config not generated',
                  style: TextStyle(color: Colors.grey)),
            ),
          const SizedBox(height: 12),
          if (!isLoading && !isGenerating)
            Row(
              children: [
                Expanded(
                  child: FilledButton.tonal(
                    onPressed: onGenerate,
                    child: const Text('Generate WARP'),
                  ),
                ),
                if (status != null && status.available) ...[
                  const SizedBox(width: 8),
                  Expanded(
                    child: OutlinedButton(
                      onPressed: onDelete,
                      style: OutlinedButton.styleFrom(
                        foregroundColor: theme.colorScheme.error,
                      ),
                      child: const Text('Delete WARP'),
                    ),
                  ),
                ],
              ],
            ),
        ],
      ),
    );
  }
}

class _WarpDetails extends StatelessWidget {
  final WarpConfig config;

  const _WarpDetails({required this.config});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _InfoRow(label: 'Endpoint', value: config.endpoint),
        if (config.clientId != null && config.clientId!.isNotEmpty)
          _InfoRow(label: 'Client ID', value: config.clientId!),
        _InfoRow(
          label: 'Latency',
          value: config.latencyMs >= 0
              ? '${config.latencyMs}ms'
              : 'Unreachable',
        ),
        _InfoRow(label: 'Protocol', value: config.protocol),
      ],
    );
  }
}

// -- Reusable Section Card --

class _SectionCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final Widget? trailing;
  final List<Widget> children;

  const _SectionCard({
    required this.icon,
    required this.title,
    this.trailing,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 20,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  title,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                if (trailing != null) trailing!,
              ],
            ),
            const SizedBox(height: 12),
            ...children,
          ],
        ),
      ),
    );
  }
}

// -- Info Row Widget --

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;

  const _InfoRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Text(
            '$label: ',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: Colors.grey),
          ),
          Expanded(
            child: Text(
              value,
              style: theme.textTheme.bodyMedium
                  ?.copyWith(fontWeight: FontWeight.w500),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}
