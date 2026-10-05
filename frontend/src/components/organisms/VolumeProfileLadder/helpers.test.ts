import { describe, expect, it } from 'vitest';
import type { VolumeProfileBin } from '@/libs/api';
import { columnPeak, profileLadderRows, profileVenue, splitPct } from './helpers';

function bin(partial: VolumeProfileBin): VolumeProfileBin {
  return partial;
}

describe('profileLadderRows', () => {
  it('keeps value-area rows and rows where OI or liquidations printed, highest price first', () => {
    const rows = profileLadderRows([
      bin({ price: '64000', volume: '10', inValueArea: false }),
      bin({ price: '65200', volume: '80', inValueArea: true, isPoc: true, oiOpened: '300000', longLiq: '80000' }),
      bin({ price: '66000', volume: '5', oiClosed: '200000' }),
      bin({ price: '67000', volume: '0', longLiq: '0', shortLiq: '0' }),
    ]);
    expect(rows.map((r) => r.price)).toEqual(['66000', '65200']);
  });
});

describe('profileVenue', () => {
  it('prefers the combined ladder', () => {
    const venue = profileVenue({
      definite: true,
      venues: [{ exchange: 'binance', bins: [{ price: '1', volume: '1' }] }],
      combined: { exchange: 'all', bins: [{ price: '2', volume: '4', oiOpened: '9' }] },
    });
    expect(venue?.exchange).toBe('all');
    expect(venue?.bins?.[0]?.oiOpened).toBe('9');
  });

  it('keeps a venue that reached the start when another venue starts later', () => {
    const venue = profileVenue({
      definite: false,
      missing: ['bybit'],
      venues: [
        { exchange: 'binance', bins: [{ price: '65200', volume: '80' }] },
        { exchange: 'bybit', bins: [{ price: '1', volume: '1' }] },
      ],
      combined: { exchange: 'all', bins: [{ price: '2', volume: '4' }] },
    });
    expect(venue?.exchange).toBe('binance');
  });
});

describe('splitPct', () => {
  it('scales buy and sell against the busiest row', () => {
    expect(splitPct('25', '75', 200)).toEqual({ left: 12.5, right: 37.5 });
    expect(columnPeak([bin({ buyVolume: '10', sellVolume: '30' }), bin({ buyVolume: '80', sellVolume: '20' })], 'buyVolume', 'sellVolume')).toBe(100);
  });
});
