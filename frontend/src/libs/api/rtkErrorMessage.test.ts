import { describe, expect, it } from 'vitest';
import { getRtkErrorStatus, rtkErrorMessage } from './rtkErrorMessage';

describe('rtkErrorMessage', () => {
  it('maps HTTP 429 via defaults when no body message', () => {
    expect(rtkErrorMessage({ status: 429, data: {} })).toMatch(/Rate limited/i);
  });

  it('maps 502 via defaults', () => {
    expect(rtkErrorMessage({ status: 502, data: {} })).toMatch(/Upstream/i);
  });

  it('prefers API error.message over default status copy', () => {
    expect(
      rtkErrorMessage({
        status: 400,
        data: { error: { code: 'bad', message: 'bad quote filter' } },
      }),
    ).toBe('bad quote filter');
  });

  it('uses resource in numeric fallback', () => {
    expect(rtkErrorMessage({ status: 418 }, { resource: 'markets' })).toBe(
      'Failed to load markets (418).',
    );
  });

  it('handles FETCH_ERROR', () => {
    expect(rtkErrorMessage({ status: 'FETCH_ERROR', error: 'TypeError: failed' })).toMatch(
      /backend/i,
    );
  });

  it('allows status message overrides (call-site wins first)', () => {
    expect(
      rtkErrorMessage(
        { status: 502, data: { error: { message: 'ignored when override set' } } },
        { statusMessages: { 502: 'Supply cache empty — try another sort.' } },
      ),
    ).toBe('Supply cache empty — try another sort.');
  });

  it('reads status helper', () => {
    expect(getRtkErrorStatus({ status: 500 })).toBe(500);
    expect(getRtkErrorStatus(null)).toBeUndefined();
  });
});
