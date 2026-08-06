import { describe, expect, it } from 'vitest';
import {
  DEFAULT_PUMP_SCAN_INTERVAL,
  DEFAULT_PUMP_SCAN_MIN_RETURN_PCT,
  DEFAULT_PUMP_SCAN_SYMBOL_LIMIT,
  PUMP_SCAN_INTERVALS,
  PUMP_SCAN_MIN_RETURN_OPTIONS,
  PUMP_SCAN_QUOTES,
} from './PumpsPage.constants';

describe('PumpsPage.constants', () => {
  it('exposes scan defaults within API bounds', () => {
    expect(DEFAULT_PUMP_SCAN_SYMBOL_LIMIT).toBeGreaterThan(0);
    expect(DEFAULT_PUMP_SCAN_SYMBOL_LIMIT).toBeLessThanOrEqual(40);
    expect(DEFAULT_PUMP_SCAN_MIN_RETURN_PCT).toBeGreaterThan(0);
    expect(PUMP_SCAN_INTERVALS).toContain(DEFAULT_PUMP_SCAN_INTERVAL);
    expect(PUMP_SCAN_QUOTES.length).toBeGreaterThan(0);
    expect(PUMP_SCAN_MIN_RETURN_OPTIONS).toContain(DEFAULT_PUMP_SCAN_MIN_RETURN_PCT);
  });
});
