import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  card: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.lg,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    padding: spacing[4],
    gap: spacing[2],
  },
  topBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  top: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: spacing[3],
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
