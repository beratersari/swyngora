import { describe, expect, it } from 'vitest';
import { buildAiChatBody } from './aiApi';

describe('buildAiChatBody', () => {
  it('trims message and keeps sessionId', () => {
    expect(
      buildAiChatBody({ message: '  hi  ', sessionId: 'mobile-ai-1' }),
    ).toEqual({ message: 'hi', sessionId: 'mobile-ai-1' });
  });
});
