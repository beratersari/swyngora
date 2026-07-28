import { describe, expect, it } from 'vitest';
import { colorValue, defaultTagForVariant, variantStyle } from './Text.helpers';

describe('Text.helpers', () => {
  it('maps variants to default tags', () => {
    expect(defaultTagForVariant('h1')).toBe('h1');
    expect(defaultTagForVariant('body')).toBe('p');
    expect(defaultTagForVariant('code')).toBe('code');
    expect(defaultTagForVariant('caption')).toBe('span');
  });

  it('resolves colors including deprecated aliases', () => {
    expect(colorValue('primary')).toBeTruthy();
    expect(colorValue('cream')).toBe(colorValue('primary'));
    expect(colorValue('steel')).toBe(colorValue('secondary'));
    expect(colorValue('error')).toBeTruthy();
  });

  it('builds variant styles with mono and weight overrides', () => {
    const style = variantStyle('body', { color: 'primary', weight: 700, mono: true });
    expect(style.fontWeight).toBe(700);
    expect(style.fontFamily).toBeTruthy();
    const overline = variantStyle('overline', { color: 'secondary' });
    expect(overline.textTransform).toBe('uppercase');
    const numeric = variantStyle('numeric', { color: 'primary' });
    expect(numeric.fontVariantNumeric).toBeTruthy();
  });
});
