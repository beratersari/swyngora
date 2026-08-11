import { describe, expect, it } from 'vitest';
import {
  canSendMessage,
  clampMessage,
  compactToolName,
  createAssistantMessage,
  createUserMessage,
  latestStepPreview,
  mergeThinkStep,
  nextProcessOpenMap,
  parseChatMarkdown,
  processPanelOpen,
  parseInlineMd,
  sanitizeChatReferences,
  sanitizeThinkingLines,
  thinkStepFromEvent,
  uniqueToolNames,
} from './AiChatPage.helpers';

describe('AiChatPage.helpers', () => {
  it('createUserMessage trims content', () => {
    const m = createUserMessage('  hello  ');
    expect(m.role).toBe('user');
    expect(m.content).toBe('hello');
    expect(m.id).toBeTruthy();
  });

  it('createAssistantMessage marks errors', () => {
    const m = createAssistantMessage('failed', { isError: true, tools: ['x'] });
    expect(m.isError).toBe(true);
    expect(m.tools).toEqual(['x']);
  });

  it('canSendMessage requires non-empty draft and not pending', () => {
    expect(canSendMessage('hi', false)).toBe(true);
    expect(canSendMessage('  ', false)).toBe(false);
    expect(canSendMessage('hi', true)).toBe(false);
  });

  it('clampMessage respects max length', () => {
    expect(clampMessage('abcdef', 4)).toBe('abcd');
    expect(clampMessage('ab', 4)).toBe('ab');
  });
});

describe('compactToolName / uniqueToolNames', () => {
  it('keeps the callable name and drops JSON dumps', () => {
    expect(compactToolName('market_agent(task=Get BTC RSI…)')).toBe('market_agent');
    expect(compactToolName('↳ get_indicators → { "rsi": 59 }')).toBe('get_indicators');
    expect(uniqueToolNames(['market_agent(task=a)', 'market_agent(task=b)', 'get_ticker'])).toEqual([
      'market_agent',
      'get_ticker',
    ]);
  });
});

describe('thinkStepFromEvent / mergeThinkStep', () => {
  it('maps live events and skips ping/final', () => {
    expect(thinkStepFromEvent({ type: 'ping' })).toBeNull();
    expect(thinkStepFromEvent({ type: 'final', reply: 'x' } as never)).toBeNull();
    const step = thinkStepFromEvent({ type: 'thinking', text: 'Need RSI' });
    expect(step?.kind).toBe('thinking');
    expect(step?.text).toBe('Need RSI');
    const merged = mergeThinkStep([step!], { ...step!, id: '2' });
    expect(merged).toHaveLength(1);
    expect(mergeThinkStep([step!], { id: '3', kind: 'tool', text: 'get_ticker' })).toHaveLength(2);
  });

  it('processPanelOpen stays live unless the user toggles', () => {
    expect(processPanelOpen(true)).toBe(true);
    expect(processPanelOpen(false)).toBe(false);
    expect(processPanelOpen(true, false)).toBe(false);
    expect(processPanelOpen(false, true)).toBe(true);
  });

  it('nextProcessOpenMap ignores default open/close and keeps user overrides', () => {
    expect(nextProcessOpenMap({}, 'a', true, true)).toEqual({});
    expect(nextProcessOpenMap({}, 'a', true, false)).toEqual({ a: false });
    expect(nextProcessOpenMap({ a: false }, 'a', false, false)).toEqual({});
    expect(nextProcessOpenMap({}, 'a', false, true)).toEqual({ a: true });
  });

  it('latestStepPreview clamps the last step', () => {
    expect(latestStepPreview([])).toBe('');
    expect(latestStepPreview([{ id: '1', kind: 'tool', text: 'get_ticker' }])).toBe('get_ticker');
    const long = 'x'.repeat(120);
    const preview = latestStepPreview([{ id: '1', kind: 'thinking', text: long }], 20);
    expect(preview.endsWith('…')).toBe(true);
    expect(preview.length).toBeLessThanOrEqual(20);
  });
});

describe('sanitizeThinkingLines', () => {
  it('drops empty, dupes, and the final reply', () => {
    expect(
      sanitizeThinkingLines(
        ['Planning…', 'Planning…', '  ', 'BTC looks interesting on the 1h.'],
        'BTC looks interesting on the 1h.',
      ),
    ).toEqual(['Planning…']);
  });
});

describe('parseChatMarkdown', () => {
  it('parses bold paragraph, list, and fence', () => {
    const blocks = parseChatMarkdown(
      '**BTCUSDT 1h RSI (14):** 59.32\n\n- Neutral zone\n- Price above EMA\n\n```\nraw\n```',
    );
    expect(blocks[0]).toEqual({ type: 'p', text: '**BTCUSDT 1h RSI (14):** 59.32' });
    expect(blocks[1]).toEqual({ type: 'ul', items: ['Neutral zone', 'Price above EMA'] });
    expect(blocks[2]).toEqual({ type: 'pre', text: 'raw' });
  });
});

describe('sanitizeChatReferences', () => {
  it('keeps http(s) urls and drops junk', () => {
    const out = sanitizeChatReferences([
      { url: 'https://coinmarketcap.com/currencies/bitcoin/', title: 'Bitcoin' },
      { url: 'javascript:alert(1)', title: 'bad' },
      { url: 'https://coinmarketcap.com/currencies/bitcoin/', title: 'dup' },
    ]);
    expect(out).toHaveLength(1);
    expect(out?.[0]?.title).toBe('Bitcoin');
  });
});

describe('parseInlineMd', () => {
  it('splits bold and code', () => {
    expect(parseInlineMd('RSI **59.32** vs `70`')).toEqual([
      { t: 'text', v: 'RSI ' },
      { t: 'strong', v: '59.32' },
      { t: 'text', v: ' vs ' },
      { t: 'code', v: '70' },
    ]);
  });
});
