import type { ChatMessage, ThinkStep, ThinkStepKind } from './AiChatPage.types';

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
  opts?: {
    tools?: string[];
    thinking?: string[];
    steps?: ChatMessage['steps'];
    streaming?: boolean;
    references?: ChatMessage['references'];
    isError?: boolean;
    id?: string;
  },
): ChatMessage {
  const thinking = opts?.thinking?.filter(Boolean);
  const steps = opts?.steps?.length
    ? opts.steps
    : stepsFromThinking(thinking);
  return {
    id: opts?.id ?? newId(),
    role: 'assistant',
    content: content.trim() || (opts?.streaming ? '' : '—'),
    tools: opts?.tools?.filter(Boolean),
    thinking,
    steps,
    streaming: opts?.streaming,
    references: sanitizeChatReferences(opts?.references),
    isError: opts?.isError,
    createdAt: Date.now(),
  };
}

const STEP_KINDS = new Set(['status', 'thinking', 'tool', 'tool_result', 'tool_error']);

export function thinkStepFromEvent(ev: {
  type?: string;
  text?: string;
  message?: string;
}): ThinkStep | null {
  const type = (ev.type || '').trim();
  if (!STEP_KINDS.has(type)) return null;
  const text = (ev.text || ev.message || '').replace(/\s+/g, ' ').trim();
  if (!text) return null;
  return {
    id: newId(),
    kind: type as ThinkStepKind,
    text: text.slice(0, 400),
  };
}

export function mergeThinkStep(prev: ThinkStep[] | undefined, next: ThinkStep): ThinkStep[] {
  const list = prev ?? [];
  const last = list[list.length - 1];
  if (last && last.kind === next.kind && last.text === next.text) return list;
  return [...list, next];
}

/** Open while streaming unless the user toggled; collapsed after the turn. */
export function processPanelOpen(streaming: boolean, userOpen?: boolean): boolean {
  if (userOpen !== undefined) return userOpen;
  return streaming;
}

/** Persist only a user override; matching the default clears it. */
export function nextProcessOpenMap(
  prev: Record<string, boolean>,
  id: string,
  streaming: boolean,
  nextOpen: boolean,
): Record<string, boolean> {
  if (nextOpen === streaming) {
    if (!(id in prev)) return prev;
    const { [id]: _removed, ...rest } = prev;
    return rest;
  }
  if (prev[id] === nextOpen) return prev;
  return { ...prev, [id]: nextOpen };
}

export function latestStepPreview(
  steps: readonly ThinkStep[] | undefined,
  max = 88,
): string {
  const last = steps?.length ? steps[steps.length - 1] : undefined;
  const text = (last?.text ?? '').replace(/\s+/g, ' ').trim();
  if (!text) return '';
  if (text.length <= max) return text;
  return `${text.slice(0, Math.max(1, max - 1)).trimEnd()}…`;
}

export function stepsFromThinking(lines: readonly string[] | undefined): ThinkStep[] | undefined {
  if (!lines?.length) return undefined;
  const out: ThinkStep[] = [];
  for (const raw of lines) {
    const text = raw.replace(/\s+/g, ' ').trim();
    if (!text) continue;
    out.push({ id: newId(), kind: 'thinking', text: text.slice(0, 400) });
  }
  return out.length ? out : undefined;
}

export function sanitizeChatReferences(
  refs: ChatMessage['references'] | undefined,
  limit = 12,
): ChatMessage['references'] {
  if (!refs?.length) return undefined;
  const out: NonNullable<ChatMessage['references']> = [];
  const seen = new Set<string>();
  for (const r of refs) {
    const url = (r.url ?? '').trim();
    if (!/^https?:\/\//i.test(url) || seen.has(url)) continue;
    seen.add(url);
    out.push({
      url,
      title: (r.title ?? '').trim() || url,
      source: r.source,
      snippet: r.snippet,
    });
    if (out.length >= limit) break;
  }
  return out.length ? out : undefined;
}

export function canSendMessage(draft: string, isPending: boolean): boolean {
  return !isPending && draft.trim().length > 0;
}

export function clampMessage(draft: string, maxLen: number): string {
  if (draft.length <= maxLen) return draft;
  return draft.slice(0, maxLen);
}

/** `market_agent(task=…)` / `↳ get_indicators → {…}` → short chip label. */
export function compactToolName(raw: string): string {
  const s = raw.trim();
  if (!s) return '';
  const m = s.match(/^(?:↳\s*)?([A-Za-z0-9_.-]+)/);
  return (m?.[1] ?? s).slice(0, 48);
}

export function uniqueToolNames(tools: readonly string[] | undefined): string[] {
  if (!tools?.length) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const t of tools) {
    const name = compactToolName(t);
    if (!name || seen.has(name)) continue;
    seen.add(name);
    out.push(name);
  }
  return out;
}

export function sanitizeThinkingLines(
  lines: readonly string[] | undefined,
  reply = '',
): string[] {
  if (!lines?.length) return [];
  const replyNorm = reply.replace(/\s+/g, ' ').trim();
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of lines) {
    const one = raw.replace(/\s+/g, ' ').trim();
    if (!one) continue;
    if (replyNorm && (one === replyNorm || replyNorm.startsWith(one) || one.startsWith(replyNorm))) {
      continue;
    }
    if (seen.has(one)) continue;
    seen.add(one);
    out.push(one);
  }
  return out;
}

export type ChatMdBlock =
  | { type: 'p'; text: string }
  | { type: 'ul'; items: string[] }
  | { type: 'ol'; items: string[] }
  | { type: 'pre'; text: string };

/** Tiny markdown subset used by the assistant (bold, lists, fenced code). */
export function parseChatMarkdown(src: string): ChatMdBlock[] {
  const text = src.replace(/\r\n/g, '\n').trim();
  if (!text) return [];
  const blocks: ChatMdBlock[] = [];
  const fence = /```[\w-]*\n?([\s\S]*?)```/g;
  let last = 0;
  let m: RegExpExecArray | null;
  const chunks: Array<{ kind: 'md' | 'pre'; text: string }> = [];
  while ((m = fence.exec(text)) != null) {
    if (m.index > last) chunks.push({ kind: 'md', text: text.slice(last, m.index) });
    chunks.push({ kind: 'pre', text: (m[1] ?? '').replace(/\n$/, '') });
    last = m.index + m[0].length;
  }
  if (last < text.length) chunks.push({ kind: 'md', text: text.slice(last) });

  for (const chunk of chunks) {
    if (chunk.kind === 'pre') {
      blocks.push({ type: 'pre', text: chunk.text });
      continue;
    }
    const lines = chunk.text.split('\n');
    let i = 0;
    while (i < lines.length) {
      const line = lines[i] ?? '';
      if (!line.trim()) {
        i += 1;
        continue;
      }
      if (/^\s*[-*]\s+/.test(line)) {
        const items: string[] = [];
        while (i < lines.length && /^\s*[-*]\s+/.test(lines[i] ?? '')) {
          items.push((lines[i] ?? '').replace(/^\s*[-*]\s+/, '').trim());
          i += 1;
        }
        blocks.push({ type: 'ul', items });
        continue;
      }
      if (/^\s*\d+\.\s+/.test(line)) {
        const items: string[] = [];
        while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i] ?? '')) {
          items.push((lines[i] ?? '').replace(/^\s*\d+\.\s+/, '').trim());
          i += 1;
        }
        blocks.push({ type: 'ol', items });
        continue;
      }
      const para: string[] = [line.trim()];
      i += 1;
      while (
        i < lines.length &&
        (lines[i] ?? '').trim() &&
        !/^\s*[-*]\s+/.test(lines[i] ?? '') &&
        !/^\s*\d+\.\s+/.test(lines[i] ?? '')
      ) {
        para.push((lines[i] ?? '').trim());
        i += 1;
      }
      blocks.push({ type: 'p', text: para.join(' ') });
    }
  }
  return blocks;
}

/** Split a line into text / bold / inline code tokens. */
export function parseInlineMd(text: string): Array<{ t: 'text' | 'strong' | 'code'; v: string }> {
  const out: Array<{ t: 'text' | 'strong' | 'code'; v: string }> = [];
  const re = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) != null) {
    if (m.index > last) out.push({ t: 'text', v: text.slice(last, m.index) });
    const tok = m[0];
    if (tok.startsWith('`')) out.push({ t: 'code', v: tok.slice(1, -1) });
    else out.push({ t: 'strong', v: tok.slice(2, -2) });
    last = m.index + tok.length;
  }
  if (last < text.length) out.push({ t: 'text', v: text.slice(last) });
  return out.length ? out : [{ t: 'text', v: text }];
}
