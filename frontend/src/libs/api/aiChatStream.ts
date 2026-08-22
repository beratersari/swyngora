import { env } from '@/config/env';
import { getBrowserApiToken } from '@/libs/utils/apiAuth';
import { getOrCreateClientId } from '@/libs/utils/clientId';
import type { AiChatResponse } from './endpoints/aiApi';

export type AiStreamEvent = {
  type: string;
  text?: string;
  reply?: string;
  tools?: string[];
  thinking?: string[];
  references?: AiChatResponse['references'];
  message?: string;
  sessionId?: string;
  note?: string;
};

export type StreamAiChatArg = {
  message: string;
  sessionId?: string;
  signal?: AbortSignal;
  onEvent: (ev: AiStreamEvent) => void;
};

function apiUrl(path: string): string {
  const base = env.apiBaseUrl.replace(/\/+$/, '');
  return `${base}${path}`;
}

/**
 * POST /api/v1/ai/chat/stream and yield NDJSON events.
 * Streaming is not modeled in RTK Query; this is the dedicated client.
 */
export async function streamAiChat(arg: StreamAiChatArg): Promise<AiStreamEvent> {
  const res = await fetch(apiUrl('/api/v1/ai/chat/stream'), {
    method: 'POST',
    headers: {
      Accept: 'application/x-ndjson, application/json',
      'Content-Type': 'application/json',
      'X-Client-Id': getOrCreateClientId(),
      ...(getBrowserApiToken() ? { Authorization: `Bearer ${getBrowserApiToken()}` } : {}),
    },
    body: JSON.stringify({
      message: arg.message,
      ...(arg.sessionId ? { sessionId: arg.sessionId } : {}),
    }),
    signal: arg.signal,
  });
  const ct = res.headers.get('content-type') || '';
  if (!res.ok) {
    const raw = await res.text();
    throw Object.assign(new Error(raw.slice(0, 300) || `AI stream ${res.status}`), {
      status: res.status,
      data: raw,
    });
  }
  if (!ct.includes('ndjson') || !res.body) {
    throw Object.assign(new Error('AI stream unavailable'), { status: res.status || 404 });
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  let finalEv: AiStreamEvent = { type: 'final', reply: '' };

  const consume = (chunk: string) => {
    buf += chunk;
    const lines = buf.split('\n');
    buf = lines.pop() ?? '';
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      let ev: AiStreamEvent;
      try {
        ev = JSON.parse(trimmed) as AiStreamEvent;
      } catch {
        continue;
      }
      if (!ev?.type || ev.type === 'ping') continue;
      arg.onEvent(ev);
      if (ev.type === 'final' && (ev.reply || ev.tools || ev.thinking || ev.references)) {
        finalEv = ev;
      }
      if (ev.type === 'error' && ev.message) {
        throw Object.assign(new Error(ev.message), { status: 502 });
      }
    }
  };

  while (true) {
    const { done, value } = await reader.read();
    if (value) consume(decoder.decode(value, { stream: !done }));
    if (done) {
      consume(decoder.decode());
      break;
    }
  }
  return finalEv;
}
