import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  wrap: {
    minWidth: 56,
    alignItems: 'flex-end',
  },
  loading: {
    opacity: 0.6,
  },
  gap: {
    marginLeft: spacing[1],
  },
});
