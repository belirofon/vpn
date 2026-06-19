import 'package:flutter/material.dart';
import '../../data/models/vpn_config.dart';

class ServerInfoCard extends StatelessWidget {
  final VpnConfig config;

  const ServerInfoCard({super.key, required this.config});

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
