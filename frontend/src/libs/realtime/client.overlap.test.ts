import { afterEach, describe, expect, it, vi } from 'vitest';
import { RealtimeClient, resetRealtimeClient } from './client';

vi.mock('./ticket', () => ({
  mintRealtimeTicket: vi.fn(),
}));

import { mintRealtimeTicket } from './ticket';

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  readyState = 1;
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  close() {
    this.readyState = 3;
    this.onclose?.();
  }
  send() {}
}

describe('RealtimeClient overlap', () => {
  afterEach(() => {
    resetRealtimeClient();
    MockWebSocket.instances = [];
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('does not open two sockets when start and subscribe overlap during ticket mint', async () => {
    let release!: (ticket: string) => void;
    const pending = new Promise<string>((resolve) => {
      release = resolve;
    });
    vi.mocked(mintRealtimeTicket).mockReturnValueOnce(pending);
    vi.stubGlobal('WebSocket', MockWebSocket);

    const client = new RealtimeClient();
    client.start();
    client.subscribePrices([{ exchange: 'binance', symbol: 'BTCUSDT' }]);
    expect(mintRealtimeTicket).toHaveBeenCalledTimes(1);

    release('ticket-a');
    await Promise.resolve();
    await Promise.resolve();

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0]?.url).toContain('ticket=ticket-a');
    expect(client.connected).toBe(false);
    MockWebSocket.instances[0]?.onopen?.();
    expect(client.connected).toBe(true);
  });

  it('ignores orphan socket close so the live socket stays connected', async () => {
    vi.mocked(mintRealtimeTicket).mockResolvedValue('ticket-a');
    vi.stubGlobal('WebSocket', MockWebSocket);
    const client = new RealtimeClient();
    client.start();
    await Promise.resolve();
    const first = MockWebSocket.instances[0];
    first?.onopen?.();
    expect(client.connected).toBe(true);

    const live = new MockWebSocket('ws://live');
    (client as unknown as { ws: MockWebSocket }).ws = live;
    first?.onclose?.();
    expect(client.connected).toBe(true);
    expect(live.readyState).toBe(1);
  });
});
