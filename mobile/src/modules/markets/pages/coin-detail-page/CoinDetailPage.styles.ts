import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  stack: {
    gap: spacing[3],
    paddingBottom: spacing[8],
  },
  retry: {
    alignSelf: 'flex-start',
  },
});
