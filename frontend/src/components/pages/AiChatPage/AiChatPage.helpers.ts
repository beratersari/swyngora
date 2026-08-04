import type { ChatMessage } from './AiChatPage.types';

function newId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `m-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function createUserMessage(content: string): ChatMessage {
  return {
    id: newId(),
    role: 'user',
    content: content.trim(),
    createdAt: Date.now(),
  };
}

export function createAssistantMessage(
  content: string,
  opts?: { tools?: string[]; thinking?: string[]; isError?: boolean },
): ChatMessage {
  return {
    id: newId(),
    role: 'assistant',
    content: content.trim() || '—',
    tools: opts?.tools?.filter(Boolean),
    thinking: opts?.thinking?.filter(Boolean),
    isError: opts?.isError,
    createdAt: Date.now(),
  };
}

export function canSendMessage(draft: string, isPending: boolean): boolean {
  return !isPending && draft.trim().length > 0;
}

export function clampMessage(draft: string, maxLen: number): string {
  if (draft.length <= maxLen) return draft;
  return draft.slice(0, maxLen);
}
