import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.md,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    padding: spacing[3],
    gap: spacing[2],
  },
  main: {
    flex: 1,
    gap: spacing[1],
  },
  top: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  symbolRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing[2],
    flexShrink: 1,
  },
  rank: {
    minWidth: 28,
  },
  topRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing[2],
  },
  bottom: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: spacing[2],
  },
  meta: {
    flex: 1,
  },
});
