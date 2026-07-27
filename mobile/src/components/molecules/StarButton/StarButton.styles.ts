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
  star: {
    fontSize: 22,
    lineHeight: 26,
  },
  starSm: {
    fontSize: 18,
    lineHeight: 22,
  },
  watched: {
    color: '#F5C542', // gold — classic favorites affordance
  },
  idle: {
    color: '#AACBC4', // pistachio
  },
});