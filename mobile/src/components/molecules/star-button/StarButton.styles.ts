import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  hit: {
    minWidth: 40,
    minHeight: 40,
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing[1],
  },
  hitSm: {
    minWidth: 36,
    minHeight: 36,
  },
});
