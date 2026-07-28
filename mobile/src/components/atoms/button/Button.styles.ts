import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  base: {
    paddingVertical: spacing[3],
    paddingHorizontal: spacing[4],
    borderRadius: radii.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  primary: {
    backgroundColor: semanticColors.action.primary,
  },
  secondary: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: semanticColors.border.default,
  },
  disabled: {
    opacity: 0.5,
  },
  label: {
    color: semanticColors.action.primaryText,
    fontWeight: '600',
    fontSize: 14,
  },
  labelSecondary: {
    color: semanticColors.text.primary,
  },
});
