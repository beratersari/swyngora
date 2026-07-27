import { describe, expect, it } from 'vitest';
import { EMA_COLORS, FALLBACK_EMA_COLORS } from './IndicatorPanel.constants';
import { emaColor } from './IndicatorPanel.helpers';

describe('emaColor', () => {
  it('uses known period colors', () => {
    expect(emaColor('12', 0)).toBe(EMA_COLORS['12']);
  });

  it('falls back by index for unknown keys', () => {
    expect(emaColor('99', 0)).toBe(FALLBACK_EMA_COLORS[0]);
    expect(emaColor('99', FALLBACK_EMA_COLORS.length)).toBe(FALLBACK_EMA_COLORS[0]);
  });
});
