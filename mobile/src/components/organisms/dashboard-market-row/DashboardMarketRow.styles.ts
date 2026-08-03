import { StyleSheet } from 'react-native';
import { semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: spacing[3],
    paddingHorizontal: spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: semanticColors.border.default,
    gap: spacing[2],
  },
  left: {
    flex: 1,
    gap: spacing[1],
  },
  right: {
    alignItems: 'flex-end',
    gap: spacing[1],
  },
  exchange: {
    textTransform: 'capitalize',
  },
});
