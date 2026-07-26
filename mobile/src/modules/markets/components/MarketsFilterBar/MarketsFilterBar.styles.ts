import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  root: {
    gap: spacing[3],
  },
  search: {
    backgroundColor: semanticColors.bg.muted,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    borderRadius: radii.md,
    paddingHorizontal: spacing[3],
    paddingVertical: spacing[3],
    color: semanticColors.text.primary,
    fontSize: 14,
  },
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing[2],
    alignItems: 'center',
  },
  label: {
    marginRight: spacing[1],
  },
  option: {
    paddingVertical: spacing[1],
    paddingHorizontal: spacing[2],
    borderRadius: radii.sm,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
    backgroundColor: semanticColors.bg.muted,
  },
  optionActive: {
    borderColor: semanticColors.border.focus,
    backgroundColor: semanticColors.action.primary,
  },
  tagsScroll: {
    maxHeight: 88,
  },
  tagsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing[1],
  },
  clearBtn: {
    paddingVertical: spacing[1],
    paddingHorizontal: spacing[2],
  },
});
