import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  headerBlock: {
    gap: spacing[3],
    marginBottom: spacing[2],
  },
  block: {
    gap: spacing[1],
  },
  back: {
    alignSelf: 'flex-start',
  },
  hint: {
    marginTop: spacing[1],
  },
});
