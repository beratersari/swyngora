import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  root: {
    gap: spacing[2],
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing[2],
  },
  search: {
    flex: 1,
  },
  filterBtn: {
    paddingVertical: spacing[3],
    paddingHorizontal: spacing[3],
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  filterBtnActive: {
    borderColor: semanticColors.border.focus,
    backgroundColor: semanticColors.action.primary,
  },
  favBtn: {
    paddingVertical: spacing[2],
    paddingHorizontal: spacing[3],
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  favBtnActive: {
    borderColor: '#F5C542',
    backgroundColor: 'rgba(245, 197, 66, 0.15)',
  },
  btnInner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing[2],
  },
});
