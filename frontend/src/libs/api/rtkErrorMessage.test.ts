import { describe, expect, it } from 'vitest';
import {
  getRtkErrorRawMessage,
  getRtkErrorStatus,
  isFetchBaseQueryError,
  isSerializedError,
  rtkErrorMessage,
} from './rtkErrorMessage';

describe('rtkErrorMessage', () => {
  it('maps HTTP status defaults when no body message', () => {
    expect(rtkErrorMessage({ status: 429, data: {} })).toMatch(/Rate limited/i);
    expect(rtkErrorMessage({ status: 502, data: {} })).toMatch(/Upstream/i);
    expect(rtkErrorMessage({ status: 400, data: {} })).toMatch(/Invalid request/i);
    expect(rtkErrorMessage({ status: 401, data: {} })).toMatch(/authorized/i);
    expect(rtkErrorMessage({ status: 403, data: {} })).toMatch(/denied|Access/i);
    expect(rtkErrorMessage({ status: 404, data: {} })).toMatch(/not found/i);
    expect(rtkErrorMessage({ status: 500, data: {} })).toMatch(/server/i);
    expect(rtkErrorMessage({ status: 503, data: {} })).toMatch(/unavailable/i);
  });

  it('maps RTK string statuses', () => {
    expect(rtkErrorMessage({ status: 'FETCH_ERROR', error: 'TypeError: failed' })).toMatch(
      /backend|reach/i,
    );
    // raw string `data` wins over default parse copy (proven API preference)
    expect(
      rtkErrorMessage({ status: 'PARSING_ERROR', data: 'raw body', originalStatus: 200 }),
    ).toBe('raw body');
    expect(rtkErrorMessage({ status: 'PARSING_ERROR', data: {} })).toMatch(/unexpected|parse/i);
    expect(rtkErrorMessage({ status: 'TIMEOUT_ERROR' })).toMatch(/timed out|timeout/i);
  });

  it('prefers API error.message over default status copy', () => {
    expect(
      rtkErrorMessage({
        status: 400,
        data: { error: { code: 'bad', message: 'bad quote filter' } },
      }),
    ).toBe('bad quote filter');
  });

  it('uses top-level data.message and string data', () => {
    expect(rtkErrorMessage({ status: 400, data: { message: 'top level' } })).toBe('top level');
    expect(rtkErrorMessage({ status: 400, data: '  plain string  ' })).toBe('plain string');
  });

  it('uses resource in numeric fallback', () => {
    expect(rtkErrorMessage({ status: 418 }, { resource: 'markets' })).toBe(
      'Failed to load markets (418).',
    );
    expect(rtkErrorMessage({ status: 418 })).toMatch(/418/);
  });

  it('generic fallbacks without status', () => {
    expect(rtkErrorMessage(null)).toMatch(/wrong|Something/i);
    expect(rtkErrorMessage({}, { resource: 'ticker' })).toMatch(/ticker/i);
  });

  it('uses SerializedError message', () => {
    expect(rtkErrorMessage({ message: 'boom' })).toBe('boom');
  });

  it('allows status message overrides (call-site wins first)', () => {
    expect(
      rtkErrorMessage(
        { status: 502, data: { error: { message: 'ignored when override set' } } },
        { statusMessages: { 502: 'Supply cache empty — try another sort.' } },
      ),
    ).toBe('Supply cache empty — try another sort.');
  });

  it('reads status and raw message helpers', () => {
    expect(getRtkErrorStatus({ status: 500 })).toBe(500);
    expect(getRtkErrorStatus({ status: 'FETCH_ERROR' })).toBe('FETCH_ERROR');
    expect(getRtkErrorStatus(null)).toBeUndefined();
    expect(getRtkErrorStatus({ status: true })).toBeUndefined();
    expect(getRtkErrorRawMessage({ data: { error: { message: ' nested ' } } })).toBe('nested');
    expect(getRtkErrorRawMessage({ data: '  hi  ' })).toBe('hi');
    expect(getRtkErrorRawMessage(null)).toBeUndefined();
  });

  it('type guards', () => {
    expect(isFetchBaseQueryError({ status: 500 })).toBe(true);
    expect(isFetchBaseQueryError(null)).toBe(false);
    expect(isSerializedError({ message: 'x' })).toBe(true);
    expect(isSerializedError({ status: 500, message: 'x' })).toBe(false);
  });
});
