import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  card: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.lg,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    overflow: 'hidden',
    padding: spacing[2],
    minHeight: 260,
  },
  host: {
    width: '100%',
    height: 260,
  },
  center: {
    minHeight: 260,
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing[4],
  },
  olderHint: {
    marginTop: 6,
  },
});
