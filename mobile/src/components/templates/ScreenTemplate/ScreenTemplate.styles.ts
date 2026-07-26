import { StyleSheet } from 'react-native';
import { semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: semanticColors.bg.canvas,
  },
  header: {
    paddingHorizontal: spacing[4],
    paddingTop: spacing[4],
    paddingBottom: spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: semanticColors.border.default,
  },
  content: {
    flex: 1,
    minHeight: 0,
    padding: spacing[4],
    gap: spacing[3],
  },
  footer: {
    padding: spacing[4],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: semanticColors.border.default,
  },
});
