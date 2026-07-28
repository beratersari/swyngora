import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  list: { flex: 1 },
  content: { gap: spacing[2], paddingBottom: spacing[6] },
  center: { padding: spacing[4], alignItems: 'center', gap: spacing[3] },
  skeletonStack: { gap: spacing[2] },
});
