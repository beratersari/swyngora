import { REALTIME_OP, REALTIME_PING_MS, REALTIME_TYPE } from './constants';
import {
  normalizeSymbolRef,
  realtimeWsUrl,
  reconnectDelayMs,
  symbolKey,
  uniqueSymbolRefs,
} from './helpers';
import { mintRealtimeTicket } from './ticket';
import type { RealtimeMessage, RealtimeSymbolRef } from './realtime.types';

type StatusListener = (connected: boolean) => void;
type MessageListener = (msg: RealtimeMessage) => void;

/**
 * One shared WebSocket. Refcounted subscribe/unsubscribe; resends
 * subscriptions after every reconnect so the session continues normally.
 */
export class RealtimeClient {
  private ws: WebSocket | null = null;
  private stopped = true;
  private opening = false;
  private openGen = 0;
  private attempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private priceCounts = new Map<string, { ref: RealtimeSymbolRef; count: number }>();
  private portfolioId: string | null = null;
  private portfolioCount = 0;
  private statusListeners = new Set<StatusListener>();
  private messageListeners = new Set<MessageListener>();
  connected = false;

  start(): void {
    if (!this.stopped && (this.ws || this.opening)) return;
    this.stopped = false;
    this.open();
  }

  stop(): void {
    this.stopped = true;
    this.clearTimers();
    this.ws?.close();
    this.ws = null;
    this.setConnected(false);
  }

  onStatus(fn: StatusListener): () => void {
    this.statusListeners.add(fn);
    fn(this.connected);
    return () => this.statusListeners.delete(fn);
  }

  onMessage(fn: MessageListener): () => void {
    this.messageListeners.add(fn);
    return () => this.messageListeners.delete(fn);
  }

  subscribePrices(symbols: RealtimeSymbolRef[]): void {
    const added: RealtimeSymbolRef[] = [];
    for (const raw of uniqueSymbolRefs(symbols)) {
      const k = symbolKey(raw);
      const cur = this.priceCounts.get(k);
      if (!cur) {
        this.priceCounts.set(k, { ref: raw, count: 1 });
        added.push(raw);
      } else {
        cur.count += 1;
      }
    }
    if (added.length && this.connected) {
      this.send({ type: REALTIME_OP.subscribePrices, symbols: added });
    }
    this.ensureStarted();
  }

  unsubscribePrices(symbols: RealtimeSymbolRef[]): void {
    const removed: RealtimeSymbolRef[] = [];
    for (const raw of uniqueSymbolRefs(symbols)) {
      const k = symbolKey(raw);
      const cur = this.priceCounts.get(k);
      if (!cur) continue;
      cur.count -= 1;
      if (cur.count <= 0) {
        this.priceCounts.delete(k);
        removed.push(raw);
      }
    }
    if (removed.length && this.connected) {
      this.send({ type: REALTIME_OP.unsubscribePrices, symbols: removed });
    }
  }

  subscribePortfolio(portfolioId: string): void {
    const id = portfolioId.trim();
    if (!id) return;
    if (this.portfolioId && this.portfolioId !== id) {
      if (this.connected) this.send({ type: REALTIME_OP.unsubscribePortfolio });
      this.portfolioCount = 0;
    }
    this.portfolioId = id;
    this.portfolioCount += 1;
    if (this.portfolioCount === 1 && this.connected) {
      this.send({ type: REALTIME_OP.subscribePortfolio, portfolioId: id });
    }
    this.ensureStarted();
  }

  unsubscribePortfolio(portfolioId?: string): void {
    const id = (portfolioId ?? this.portfolioId ?? '').trim();
    if (!id || this.portfolioId !== id) return;
    this.portfolioCount = Math.max(0, this.portfolioCount - 1);
    if (this.portfolioCount === 0) {
      this.portfolioId = null;
      if (this.connected) this.send({ type: REALTIME_OP.unsubscribePortfolio });
    }
  }

  private ensureStarted(): void {
    if (this.stopped) this.start();
    else if (!this.ws && !this.reconnectTimer && !this.opening) this.open();
  }

  private open(): void {
    if (this.stopped || this.opening) return;
    this.clearTimers();
    this.opening = true;
    void this.openWithTicket();
  }

  private async openWithTicket(): Promise<void> {
    const gen = ++this.openGen;
    try {
      const ticket = await mintRealtimeTicket();
      if (this.stopped || gen !== this.openGen) return;
      const url = realtimeWsUrl(undefined, undefined, undefined, undefined, ticket);
      let ws: WebSocket;
      try {
        ws = new WebSocket(url);
      } catch {
        this.scheduleReconnect();
        return;
      }
      if (this.ws && this.ws !== ws) {
        const prev = this.ws;
        this.ws = null;
        prev.close();
      }
      this.ws = ws;
      ws.onopen = () => {
        if (this.ws !== ws) return;
        this.attempt = 0;
        this.setConnected(true);
        this.resubscribeAll();
        this.pingTimer = setInterval(() => {
          this.send({ type: REALTIME_OP.ping });
        }, REALTIME_PING_MS);
      };
      ws.onmessage = (ev) => {
        if (this.ws !== ws) return;
        let parsed: unknown = ev.data;
        if (typeof ev.data === 'string') {
          try {
            parsed = JSON.parse(ev.data);
          } catch {
            return;
          }
        }
        if (!parsed || typeof parsed !== 'object' || !('type' in parsed)) return;
        const msg = parsed as RealtimeMessage;
        if (msg.type === REALTIME_TYPE.error || msg.type === REALTIME_TYPE.hello || msg.type === REALTIME_TYPE.ack) {
          // still fan out
        }
        for (const fn of this.messageListeners) fn(msg);
      };
      ws.onerror = () => {
        /* onclose handles reconnect */
      };
      ws.onclose = () => {
        if (this.ws !== ws) return;
        this.ws = null;
        this.setConnected(false);
        this.clearPing();
        if (!this.stopped) this.scheduleReconnect();
      };
    } finally {
      if (gen === this.openGen) this.opening = false;
    }
  }

  private resubscribeAll(): void {
    const symbols = [...this.priceCounts.values()].map((v) => v.ref);
    if (symbols.length) {
      this.send({ type: REALTIME_OP.subscribePrices, symbols });
    }
    if (this.portfolioId) {
      this.send({ type: REALTIME_OP.subscribePortfolio, portfolioId: this.portfolioId });
    }
  }

  private send(body: unknown): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    try {
      this.ws.send(JSON.stringify(body));
    } catch {
      /* ignore */
    }
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.reconnectTimer) return;
    const delay = reconnectDelayMs(this.attempt);
    this.attempt += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, delay);
  }

  private setConnected(next: boolean): void {
    if (this.connected === next) return;
    this.connected = next;
    for (const fn of this.statusListeners) fn(next);
  }

  private clearPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  private clearTimers(): void {
    this.clearPing();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

let singleton: RealtimeClient | null = null;

export function getRealtimeClient(): RealtimeClient {
  if (!singleton) singleton = new RealtimeClient();
  return singleton;
}

/** Test-only. */
export function resetRealtimeClient(): void {
  singleton?.stop();
  singleton = null;
}

export function normalizeRefsForTest(refs: RealtimeSymbolRef[]): RealtimeSymbolRef[] {
  return refs.map((r) => normalizeSymbolRef(r)).filter((x): x is RealtimeSymbolRef => Boolean(x));
}
