import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  list: {
    flex: 1,
  },
  content: {
    gap: spacing[2],
    paddingBottom: spacing[6],
  },
  center: {
    paddingVertical: spacing[6],
    gap: spacing[3],
    alignItems: 'center',
  },
  skeletonStack: {
    gap: spacing[2],
  },
});
