export type AiContextParams = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  draft?: string;
};

/**
 * Build an editable user draft from navigation context.
 * Prefer explicit `draft`; otherwise synthesize from market context.
 */
export function buildContextPrompt(params: AiContextParams): string {
  const draft = params.draft?.trim();
  if (draft) return draft;

  const symbol = params.symbol?.trim().toUpperCase() ?? '';
  const exchange = params.exchange?.trim().toLowerCase() ?? '';
  const interval = params.interval?.trim() ?? '';

  if (!symbol && !exchange) return '';

  const parts: string[] = [];
  if (symbol && exchange) {
    parts.push(`Explain ${symbol} on ${exchange}`);
  } else if (symbol) {
    parts.push(`Explain ${symbol}`);
  } else if (exchange) {
    parts.push(`Explain markets on ${exchange}`);
  }
  if (interval) {
    parts.push(`(${interval} interval)`);
  }
  parts.push('— latest price, 24h change, and RSI if available. Not financial advice.');
  return parts.join(' ');
}

export type ChatRole = 'user' | 'assistant' | 'system';

export type ChatMessageModel = {
  id: string;
  role: ChatRole;
  text: string;
  tools?: string[];
  thinking?: string[];
  createdAt: number;
  /** True while waiting for assistant (placeholder bubble). */
  pending?: boolean;
  /** Soft error attached to a user turn. */
  error?: string | null;
};

export function createMessageId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `msg-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

export function createUserMessage(text: string): ChatMessageModel {
  return {
    id: createMessageId(),
    role: 'user',
    text: text.trim(),
    createdAt: Date.now(),
  };
}

export function createAssistantMessage(
  text: string,
  opts?: { tools?: string[]; thinking?: string[] },
): ChatMessageModel {
  return {
    id: createMessageId(),
    role: 'assistant',
    text,
    tools: opts?.tools,
    thinking: opts?.thinking,
    createdAt: Date.now(),
  };
}

export function createPendingAssistantMessage(): ChatMessageModel {
  return {
    id: createMessageId(),
    role: 'assistant',
    text: '',
    pending: true,
    createdAt: Date.now(),
  };
}

/** Keep only the last `max` messages. */
export function trimMessages(
  messages: ChatMessageModel[],
  max: number,
): ChatMessageModel[] {
  if (messages.length <= max) return messages;
  return messages.slice(messages.length - max);
}
