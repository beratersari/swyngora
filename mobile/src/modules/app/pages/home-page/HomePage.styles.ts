import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  scroll: {
    flexGrow: 1,
    paddingBottom: spacing[6],
    gap: spacing[1],
  },
  intro: {
    marginBottom: spacing[3],
  },
  quick: {
    marginBottom: spacing[3],
  },
  footerCard: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.lg,
    padding: spacing[4],
    gap: spacing[2],
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    marginTop: spacing[2],
  },
  row: {
    gap: spacing[1],
  },
  badgeOk: {
    color: semanticColors.status.success,
  },
  badgeError: {
    color: semanticColors.status.error,
  },
});
