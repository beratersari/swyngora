import { describe, expect, it } from 'vitest';
import { rtkErrorMessage } from './rtkErrorMessage';

describe('rtkErrorMessage', () => {
  it('uses resource fallback when error is empty', () => {
    expect(rtkErrorMessage(null, { resource: 'markets' })).toBe('Failed to load markets');
    expect(rtkErrorMessage(undefined)).toBe('Request failed');
  });

  it('maps FETCH_ERROR to network message', () => {
    expect(rtkErrorMessage({ status: 'FETCH_ERROR' })).toBe(
      'Network error — is the backend running?',
    );
  });

  it('reads nested API error message', () => {
    expect(
      rtkErrorMessage({
        data: { error: { message: 'invalid clientId' } },
      }),
    ).toBe('invalid clientId');
  });

  it('reads data.message', () => {
    expect(rtkErrorMessage({ data: { message: 'bad request' } })).toBe('bad request');
  });

  it('reads top-level error string', () => {
    expect(rtkErrorMessage({ error: 'timeout' })).toBe('timeout');
  });

  it('reads Error.message', () => {
    expect(rtkErrorMessage({ message: 'boom' })).toBe('boom');
  });

  it('formats numeric status', () => {
    expect(rtkErrorMessage({ status: 502 })).toBe('Request failed (502)');
  });
});
