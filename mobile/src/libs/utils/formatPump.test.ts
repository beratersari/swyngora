import { describe, expect, it } from 'vitest';
import {
  formatPumpEventTime,
  formatPumpReturnPct,
  formatVolumeRatio,
  pumpModeLabel,
  pumpReturnTone,
} from './formatPump';

describe('formatPumpReturnPct', () => {
  it('formats signed percent', () => {
    expect(formatPumpReturnPct(12.345)).toBe('+12.35%');
    expect(formatPumpReturnPct(-6.1)).toBe('-6.10%');
    expect(formatPumpReturnPct(null)).toBe('—');
  });
});

describe('pumpReturnTone', () => {
  it('maps sign to tone', () => {
    expect(pumpReturnTone(5)).toBe('success');
    expect(pumpReturnTone(-3)).toBe('error');
  });
});

describe('formatVolumeRatio', () => {
  it('formats or empty', () => {
    expect(formatVolumeRatio(2.14)).toBe('vol ×2.1');
    expect(formatVolumeRatio(0)).toBe('');
  });
});

describe('pumpModeLabel', () => {
  it('labels modes', () => {
    expect(pumpModeLabel('close_return')).toBe('Close return');
  });
});

describe('formatPumpEventTime', () => {
  it('handles invalid', () => {
    expect(formatPumpEventTime(undefined)).toBe('—');
  });
});
