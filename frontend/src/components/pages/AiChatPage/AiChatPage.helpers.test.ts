import { describe, expect, it } from 'vitest';
import {
  canSendMessage,
  clampMessage,
  createAssistantMessage,
  createUserMessage,
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
