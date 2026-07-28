import { StyleSheet } from 'react-native';
import { spacing } from '@/styles/tokens';

export const styles = StyleSheet.create({
  root: {
    marginTop: spacing[1],
    marginBottom: spacing[2],
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing[1],
    alignItems: 'center',
  },
  label: {
    marginRight: spacing[1],
  },
});
