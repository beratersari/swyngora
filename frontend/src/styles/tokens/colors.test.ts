import { describe, expect, it } from 'vitest';
import { financeColors, palette, semanticColors, withAlpha } from './colors';

describe('withAlpha', () => {
  it('converts brand hex to rgba', () => {
    expect(withAlpha(financeColors.amber, 0.5)).toBe('rgba(240, 162, 2, 0.5)');
    expect(withAlpha('#F0A202', 1)).toBe('rgba(240, 162, 2, 1)');
  });

  it('clamps alpha', () => {
    expect(withAlpha('#FFFFFF', 2)).toBe('rgba(255, 255, 255, 1)');
    expect(withAlpha('#FFFFFF', -1)).toBe('rgba(255, 255, 255, 0)');
  });
});

describe('semanticColors role rules', () => {
  it('uses amber for chrome accents and teal/red only for market direction', () => {
    expect(semanticColors.accent.default).toBe(financeColors.amber);
    expect(semanticColors.action.primary).toBe(financeColors.amber);
    expect(semanticColors.chart.up).toBe(financeColors.up);
    expect(semanticColors.chart.down).toBe(financeColors.down);
    expect(semanticColors.chart.up).not.toBe(semanticColors.accent.default);
    expect(semanticColors.border.focus).toBe(financeColors.amber);
  });

  it('uses silver/mist for muted text (not ash)', () => {
    expect(semanticColors.text.secondary).toBe(financeColors.silver);
    expect(semanticColors.text.tertiary).toBe(financeColors.mist);
    expect(semanticColors.text.tertiary).not.toBe(financeColors.ash);
  });

  it('maps charcoal surface hierarchy', () => {
    expect(semanticColors.bg.canvas).toBe(financeColors.ink);
    expect(semanticColors.bg.page).toBe(financeColors.graphite);
    expect(semanticColors.bg.chrome).toBe(financeColors.steel);
    expect(semanticColors.bg.elevated).toBe(financeColors.pewter);
    expect(semanticColors.bg.tableHeader).toBe(financeColors.steel);
  });

  it('keeps legacy palette keys pointed at the finance hexes', () => {
    expect(palette.bangladeshGreen).toBe(financeColors.amber);
    expect(palette.richBlack).toBe(financeColors.ink);
    expect(palette.caribbeanGreen).toBe(financeColors.up);
  });
});
