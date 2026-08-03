import { describe, expect, it } from 'vitest';
import {
  buildContextPrompt,
  createAssistantMessage,
  createUserMessage,
  trimMessages,
} from './aiContextPrompt';

describe('buildContextPrompt', () => {
  it('prefers explicit draft', () => {
    expect(
      buildContextPrompt({
        draft: '  custom q  ',
        symbol: 'BTCUSDT',
        exchange: 'binance',
      }),
    ).toBe('custom q');
  });

  it('builds from symbol + exchange + interval', () => {
    const text = buildContextPrompt({
      symbol: 'btcusdt',
      exchange: 'Binance',
      interval: '1h',
    });
    expect(text).toMatch(/BTCUSDT/);
    expect(text).toMatch(/binance/);
    expect(text).toMatch(/1h/);
    expect(text.toLowerCase()).toMatch(/not financial advice/);
  });

  it('returns empty without context', () => {
    expect(buildContextPrompt({})).toBe('');
  });
});

describe('chat message helpers', () => {
  it('creates user and assistant messages', () => {
    const u = createUserMessage(' hello ');
    expect(u.role).toBe('user');
    expect(u.text).toBe('hello');
    const a = createAssistantMessage('reply', { tools: ['t1'] });
    expect(a.role).toBe('assistant');
    expect(a.tools).toEqual(['t1']);
  });

  it('trims long transcripts', () => {
    const msgs = Array.from({ length: 5 }, (_, i) =>
      createUserMessage(`m${i}`),
    );
    const trimmed = trimMessages(msgs, 3);
    expect(trimmed).toHaveLength(3);
    expect(trimmed[0]?.text).toBe('m2');
  });
});
