import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  scroll: {
    paddingBottom: spacing[8],
    gap: spacing[4],
  },
  searchBlock: {
    gap: spacing[1],
  },
  errorBlock: {
    gap: spacing[2],
  },
  headerActions: {
    marginBottom: spacing[2],
  },
});
