import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: spacing[2],
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    backgroundColor: semanticColors.bg.muted,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    borderRadius: radii.md,
    paddingHorizontal: spacing[3],
    paddingVertical: spacing[3],
    color: semanticColors.text.primary,
    fontSize: 14,
  },
  sendWrap: {
    flexShrink: 0,
  },
});
