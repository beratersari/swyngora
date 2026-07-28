import { StyleSheet } from 'react-native';
import { radii, semanticColors, spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  row: {
    marginBottom: spacing[3],
    maxWidth: '92%',
  },
  rowUser: {
    alignSelf: 'flex-end',
  },
  rowAssistant: {
    alignSelf: 'flex-start',
  },
  bubble: {
    borderRadius: radii.lg,
    paddingHorizontal: spacing[3],
    paddingVertical: spacing[3],
  },
  bubbleUser: {
    backgroundColor: semanticColors.bg.elevated,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
  },
  bubbleAssistant: {
    backgroundColor: semanticColors.bg.muted,
    borderWidth: 1,
    borderColor: semanticColors.border.default,
  },
  bubbleSystem: {
    backgroundColor: 'transparent',
    borderWidth: 0,
    paddingHorizontal: 0,
  },
  meta: {
    marginTop: spacing[1],
  },
  error: {
    marginTop: spacing[1],
  },
});
