import { StyleSheet } from 'react-native';
import { fontFamilies, semanticColors, typeScale } from '@/styles/tokens';
import type { TextColor, TypeVariant } from '@/styles/tokens';

export function textStyle(variant: TypeVariant, color: TextColor) {
  const scale = typeScale[variant];
  return {
    fontFamily: variant === 'code' || variant === 'numeric' ? fontFamilies.mono : fontFamilies.sans,
    fontSize: scale.fontSize,
    lineHeight: scale.lineHeight,
    fontWeight: scale.fontWeight,
    color: resolveColor(color),
  } as const;
}

function resolveColor(color: TextColor): string {
  switch (color) {
    case 'primary':
    case 'cream':
      return semanticColors.text.primary;
    case 'secondary':
    case 'steel':
      return semanticColors.text.secondary;
    case 'inverse':
      return semanticColors.text.inverse;
    case 'success':
      return semanticColors.status.success;
    case 'warning':
      return semanticColors.status.warning;
    case 'error':
      return semanticColors.status.error;
    default:
      return semanticColors.text.primary;
  }
}

export const styles = StyleSheet.create({
  base: {
    fontFamily: fontFamilies.sans,
  },
});
