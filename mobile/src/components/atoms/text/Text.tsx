import { Text as RNText } from 'react-native';
import type { TextProps } from './Text.types';
import { styles, textStyle } from './Text.styles';

export function Text({
  variant = 'body',
  color = 'primary',
  style,
  children,
  ...rest
}: TextProps) {
  return (
    <RNText style={[styles.base, textStyle(variant, color), style]} {...rest}>
      {children}
    </RNText>
  );
}
