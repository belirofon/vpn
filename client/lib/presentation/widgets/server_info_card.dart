import 'package:flutter/material.dart';
import '../../data/models/vpn_config.dart';

class ServerInfoCard extends StatelessWidget {
  final VpnConfig config;
  final int currentIndex;
  final int totalCount;
  final VoidCallback? onPrev;
  final VoidCallback? onNext;

  const ServerInfoCard({
    super.key,
    required this.config,
    this.currentIndex = 0,
    this.totalCount = 1,
    this.onPrev,
    this.onNext,
  });

  String _countryFlag(String code) {
    if (code.length != 2) return code;
    final first = code.codeUnitAt(0) - 0x41 + 0x1F1E6;
    final second = code.codeUnitAt(1) - 0x41 + 0x1F1E6;
    return String.fromCharCodes([first, second]);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasPrev = currentIndex > 0;
    final hasNext = currentIndex < totalCount - 1;

    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.primaryContainer,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    '${_countryFlag(config.country)}  ${config.country}',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: _latencyColor(config.latencyMs),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.speed, size: 16, color: Colors.white),
                      const SizedBox(width: 4),
                      Text(
                        '${config.latencyMs}ms',
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                          color: Colors.white,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            SizedBox(
              width: double.infinity,
              child: Text(
                config.name,
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w500,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            if (config.protocol.isNotEmpty) ...[
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: [
                  _tag(theme, config.protocol.toUpperCase(), Colors.blue),
                  if (config.tls == 'tls')
                    _tag(theme, 'TLS', Colors.green),
                  if (config.tls == 'reality')
                    _tag(theme, 'REALITY', Colors.orange),
                  if (config.network == 'ws')
                    _tag(theme, 'WS', Colors.purple),
                ],
              ),
            ],
            const SizedBox(height: 12),
            Row(
              children: [
                SizedBox(
                  width: 48,
                  height: 40,
                  child: FilledButton.tonal(
                    onPressed: hasPrev ? onPrev : null,
                    style: FilledButton.styleFrom(
                      padding: EdgeInsets.zero,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                    child: const Icon(Icons.chevron_left, size: 22),
                  ),
                ),
                Expanded(
                  child: Center(
                    child: Text(
                      '${currentIndex + 1} / $totalCount',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: Colors.grey,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
                ),
                SizedBox(
                  width: 48,
                  height: 40,
                  child: FilledButton.tonal(
                    onPressed: hasNext ? onNext : null,
                    style: FilledButton.styleFrom(
                      padding: EdgeInsets.zero,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                    child: const Icon(Icons.chevron_right, size: 22),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _tag(ThemeData theme, String label, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.bold,
          color: color,
        ),
      ),
    );
  }

  Color _latencyColor(int ms) {
    if (ms < 50) return Colors.green;
    if (ms < 100) return Colors.orange;
    return Colors.red;
  }
}
