import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  list: {
    flexGrow: 1,
    paddingVertical: spacing[2],
  },
  empty: {
    paddingVertical: spacing[6],
    gap: spacing[2],
  },
  banner: {
    marginBottom: spacing[3],
  },
  tools: {
    marginBottom: spacing[2],
    marginLeft: spacing[1],
  },
});
