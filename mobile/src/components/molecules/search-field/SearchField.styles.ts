import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  input: {
    backgroundColor: semanticColors.bg.muted,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    borderRadius: radii.md,
    paddingHorizontal: spacing[3],
    paddingVertical: spacing[3],
    color: semanticColors.text.primary,
    fontSize: 14,
  },
});
