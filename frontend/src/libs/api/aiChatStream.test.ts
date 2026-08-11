import { describe, expect, it, vi, afterEach } from 'vitest';
import { streamAiChat, type AiStreamEvent } from './aiChatStream';

describe('streamAiChat', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('parses NDJSON events and returns final', async () => {
    const body = [
      '{"type":"status","text":"Planning…"}',
      '{"type":"ping"}',
      '{"type":"thinking","text":"Need RSI"}',
      '{"type":"final","reply":"RSI 55","thinking":["Need RSI"]}',
      '{"type":"done"}',
      '',
    ].join('\n');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => 'application/x-ndjson' },
        body: {
          getReader: () => {
            const enc = new TextEncoder();
            let sent = false;
            return {
              read: async () => {
                if (sent) return { done: true, value: undefined };
                sent = true;
                return { done: false, value: enc.encode(body) };
              },
            };
          },
        },
      }),
    );
    const seen: string[] = [];
    const final = await streamAiChat({
      message: 'rsi?',
      sessionId: 's',
      onEvent: (ev: AiStreamEvent) => seen.push(ev.type),
    });
    expect(seen).toEqual(['status', 'thinking', 'final', 'done']);
    expect(final.reply).toBe('RSI 55');
  });
});
