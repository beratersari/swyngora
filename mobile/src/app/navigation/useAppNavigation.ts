import { useNavigation } from '@react-navigation/native';
import type { NavigationProp } from '@react-navigation/native';
import type { RootStackParamList } from './types';

export function useAppNavigation() {
  return useNavigation<NavigationProp<RootStackParamList>>();
}
