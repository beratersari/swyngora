import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  card: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.lg,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    padding: spacing[3],
    marginBottom: spacing[3],
    gap: spacing[2],
  },
  item: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: spacing[2],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: semanticColors.border.default,
  },
  left: { flex: 1, gap: 2 },
  right: { alignItems: 'flex-end' },
});
