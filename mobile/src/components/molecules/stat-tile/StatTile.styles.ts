import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  tile: {
    flexGrow: 1,
    flexBasis: '45%',
    minWidth: 120,
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.md,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    padding: spacing[3],
    gap: spacing[1],
  },
});
