import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing[2],
  },
  chip: {
    paddingVertical: spacing[2],
    paddingHorizontal: spacing[3],
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  chipActive: {
    backgroundColor: semanticColors.action.primary,
    borderColor: semanticColors.action.primary,
  },
  chipLabel: {
    textTransform: 'capitalize',
  },
});
