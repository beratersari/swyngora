import type { ReactNode } from 'react';
import type { TextProps as RNTextProps } from 'react-native';
import type { TextColor, TypeVariant } from '@/styles/tokens';

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  color?: TextColor;
  children: ReactNode;
};
