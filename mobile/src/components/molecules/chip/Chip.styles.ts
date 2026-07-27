import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  base: {
    paddingVertical: spacing[2],
    paddingHorizontal: spacing[3],
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  pill: {
    borderRadius: radii.pill,
  },
  box: {
    borderRadius: radii.sm,
  },
  active: {
    borderColor: semanticColors.border.focus,
    backgroundColor: semanticColors.action.primary,
  },
  labelCapitalize: {
    textTransform: 'capitalize',
  },
});
