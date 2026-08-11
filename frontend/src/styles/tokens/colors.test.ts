import { describe, expect, it } from 'vitest';
import { palette, semanticColors, withAlpha } from './colors';

describe('withAlpha', () => {
  it('converts brand hex to rgba', () => {
    expect(withAlpha(palette.mountainMeadow, 0.5)).toBe('rgba(79, 212, 165, 0.5)');
    expect(withAlpha('#03624C', 1)).toBe('rgba(3, 98, 76, 1)');
  });

  it('clamps alpha', () => {
    expect(withAlpha('#FFFFFF', 2)).toBe('rgba(255, 255, 255, 1)');
    expect(withAlpha('#FFFFFF', -1)).toBe('rgba(255, 255, 255, 0)');
  });
});

describe('semanticColors role rules', () => {
  it('keeps caribbean green off chrome accents', () => {
    expect(semanticColors.accent.default).toBe(palette.mountainMeadow);
    expect(semanticColors.chart.up).toBe(palette.caribbeanGreen);
    expect(semanticColors.border.focus).toBe(palette.caribbeanGreen);
  });

  it('uses pistachio/sage for readable muted text (not stone)', () => {
    expect(semanticColors.text.secondary).toBe(palette.pistachio);
    expect(semanticColors.text.tertiary).toBe(palette.sage);
    expect(semanticColors.text.tertiary).not.toBe(palette.stone);
  });

  it('maps surface hierarchy', () => {
    expect(semanticColors.bg.canvas).toBe(palette.richBlack);
    expect(semanticColors.bg.page).toBe(palette.darkGreen);
    expect(semanticColors.bg.chrome).toBe(palette.pine);
    expect(semanticColors.bg.elevated).toBe(palette.basil);
  });
});
