import type { CSSProperties } from 'react';
import type { ElementType } from 'react';
import { semanticColors, typeScale, type TextColor, type TypeVariant } from '@/styles/tokens';

const VARIANT_DEFAULT_TAG: Record<TypeVariant, ElementType> = {
  display: 'h1',
  h1: 'h1',
  h2: 'h2',
  h3: 'h3',
  h4: 'h4',
  bodyLg: 'p',
  body: 'p',
  bodySm: 'p',
  caption: 'span',
  overline: 'span',
  label: 'span',
  code: 'code',
  numeric: 'span',
};

const COLOR_MAP: Record<TextColor, string> = {
  primary: semanticColors.text.primary,
  secondary: semanticColors.text.secondary,
  inverse: semanticColors.text.inverse,
  cream: '#EAE0CF',
  steel: '#7288AE',
  success: semanticColors.status.success,
  warning: semanticColors.status.warning,
  error: semanticColors.status.error,
};

export function defaultTagForVariant(variant: TypeVariant): ElementType {
  return VARIANT_DEFAULT_TAG[variant];
}

export function colorValue(color: TextColor): string {
  return COLOR_MAP[color];
}

export function variantStyle(
  variant: TypeVariant,
  options: { weight?: number; mono?: boolean; color: TextColor },
): CSSProperties {
  const scale = typeScale[variant];
  const style: CSSProperties = {
    margin: 0,
    fontSize: scale.fontSize,
    lineHeight: scale.lineHeight,
    fontWeight: options.weight ?? scale.fontWeight,
    letterSpacing: 'letterSpacing' in scale ? scale.letterSpacing : undefined,
    color: colorValue(options.color),
    fontFamily:
      'fontFamily' in scale && scale.fontFamily
        ? scale.fontFamily
        : options.mono
          ? typeScale.code.fontFamily
          : undefined,
  };

  if ('textTransform' in scale && scale.textTransform) {
    style.textTransform = scale.textTransform;
  }
  if ('fontVariantNumeric' in scale && scale.fontVariantNumeric) {
    style.fontVariantNumeric = scale.fontVariantNumeric;
  }

  return style;
}
