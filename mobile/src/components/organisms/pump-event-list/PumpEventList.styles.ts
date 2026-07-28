import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  section: { gap: spacing[2] },
  card: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.md,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    padding: spacing[3],
    gap: spacing[1],
  },
  rowTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
});
