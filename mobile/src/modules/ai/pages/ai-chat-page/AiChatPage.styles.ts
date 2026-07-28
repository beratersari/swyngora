import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  root: {
    flex: 1,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    paddingBottom: spacing[3],
  },
  footer: {
    gap: spacing[2],
    paddingTop: spacing[2],
  },
  actions: {
    flexDirection: 'row',
    gap: spacing[2],
    flexWrap: 'wrap',
  },
});
