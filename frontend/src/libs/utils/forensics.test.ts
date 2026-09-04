import { describe, expect, it } from 'vitest';
import {
  forensicsPhaseById,
  forensicsVenueOptions,
  formatForensicsTime,
  pickForensicsVenue,
} from './forensics';
import type { ForensicsTape } from '@/libs/types/forensics.types';

const tape: ForensicsTape = {
  combined: {
    exchange: 'combined',
    phases: [{ phase: 'before', complete: true }],
  },
  venues: [
    { exchange: 'binance', phases: [{ phase: 'during', complete: true }] },
    { exchange: 'bybit', phases: [{ phase: 'after', complete: true }] },
  ],
};

describe('forensics helpers', () => {
  it('picks combined then named venue then first', () => {
    expect(pickForensicsVenue(tape, 'combined')?.exchange).toBe('combined');
    expect(pickForensicsVenue(tape, 'bybit')?.exchange).toBe('bybit');
    expect(pickForensicsVenue({ venues: tape.venues }, 'missing')?.exchange).toBe('binance');
  });

  it('lists venue options and finds a phase', () => {
    expect(forensicsVenueOptions(tape).map((v) => v.value)).toEqual([
      'combined',
      'binance',
      'bybit',
    ]);
    expect(forensicsPhaseById(tape.combined, 'before')?.phase).toBe('before');
  });

  it('formats times or a dash', () => {
    expect(formatForensicsTime(undefined)).toBe('—');
    expect(formatForensicsTime('2026-08-20T14:00:00.000Z')).toBe('2026-08-20 14:00:00 UTC');
  });
});
