import { describe, expect, it } from 'vitest';
import { financeColors, palette, semanticColors, withAlpha } from './colors';

describe('withAlpha', () => {
  it('converts brand hex to rgba', () => {
    expect(withAlpha(financeColors.amber, 0.5)).toBe('rgba(56, 97, 251, 0.5)');
    expect(withAlpha('#3861FB', 1)).toBe('rgba(56, 97, 251, 1)');
  });

  it('clamps alpha', () => {
    expect(withAlpha('#FFFFFF', 2)).toBe('rgba(255, 255, 255, 1)');
    expect(withAlpha('#FFFFFF', -1)).toBe('rgba(255, 255, 255, 0)');
  });
});

describe('semanticColors role rules', () => {
  it('uses blue for chrome and green/red only for market direction', () => {
    expect(semanticColors.accent.default).toBe(financeColors.amber);
    expect(semanticColors.action.primary).toBe('#3861FB');
    expect(semanticColors.chart.up).toBe('#16C784');
    expect(semanticColors.chart.down).toBe('#EA3943');
    expect(semanticColors.chart.up).not.toBe(semanticColors.accent.default);
    expect(semanticColors.border.focus).toBe(financeColors.amber);
  });

  it('uses silver/mist for muted text', () => {
    expect(semanticColors.text.secondary).toBe(financeColors.silver);
    expect(semanticColors.text.tertiary).toBe(financeColors.mist);
  });

  it('maps a light consumer surface hierarchy', () => {
    expect(semanticColors.bg.canvas).toBe('#FFFFFF');
    expect(semanticColors.bg.page).toBe('#F8FAFD');
    expect(semanticColors.bg.chrome).toBe('#FFFFFF');
    expect(semanticColors.text.primary).toBe('#0D1421');
  });

  it('keeps legacy palette keys pointed at the new hexes', () => {
    expect(palette.bangladeshGreen).toBe(financeColors.amber);
    expect(palette.caribbeanGreen).toBe(financeColors.up);
  });
});
