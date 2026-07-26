import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  footer: {
    flexDirection: 'row',
    gap: spacing[2],
  },
  footerBtn: {
    flex: 1,
  },
});
