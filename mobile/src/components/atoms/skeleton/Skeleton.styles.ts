import { StyleSheet } from 'react-native';
import { radii, semanticColors } from '@/styles/tokens';

export const styles = StyleSheet.create({
  base: {
    backgroundColor: semanticColors.skeleton.base,
    borderRadius: radii.md,
    overflow: 'hidden',
  },
});
