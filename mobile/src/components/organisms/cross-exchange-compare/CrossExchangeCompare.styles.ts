import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  root: {
    gap: spacing[2],
  },
  list: {
    gap: spacing[1],
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: spacing[3],
    paddingHorizontal: spacing[3],
    borderRadius: radii.md,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  rowSource: {
    borderColor: semanticColors.border.focus,
  },
  rowCheapest: {
    borderColor: semanticColors.status.success,
  },
  left: {
    flex: 1,
    gap: spacing[1],
    paddingRight: spacing[2],
  },
  right: {
    alignItems: 'flex-end',
    gap: spacing[1],
  },
  badges: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing[1],
  },
  badge: {
    paddingHorizontal: spacing[2],
    paddingVertical: 2,
    borderRadius: radii.sm,
    backgroundColor: semanticColors.action.primary,
  },
  badgeMuted: {
    backgroundColor: semanticColors.bg.muted,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
  },
});
