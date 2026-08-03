import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  card: {
    backgroundColor: semanticColors.bg.muted,
    borderRadius: radii.lg,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: semanticColors.border.default,
    overflow: 'hidden',
    paddingTop: spacing[3],
    paddingHorizontal: spacing[3],
    paddingBottom: spacing[1],
    marginBottom: spacing[3],
  },
  body: {
    minHeight: 48,
  },
  empty: {
    paddingVertical: spacing[4],
  },
  error: {
    paddingVertical: spacing[2],
    gap: spacing[2],
  },
  skeletons: {
    gap: spacing[2],
    paddingBottom: spacing[3],
  },
});
