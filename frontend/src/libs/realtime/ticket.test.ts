import { describe, expect, it, vi } from 'vitest';
import { mintRealtimeTicket } from './ticket';

describe('mintRealtimeTicket', () => {
  it('POSTs the session Authorization header and returns the ticket', async () => {
    const fetchImpl = vi.fn(async () =>
      new Response(JSON.stringify({ ticket: 'abc123', clientId: 'c1' }), { status: 200 }),
    );
    const got = await mintRealtimeTicket(fetchImpl as unknown as typeof fetch, 'https://api.example.com', 'c1', 'swy_user');
    expect(got).toBe('abc123');
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImpl.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('https://api.example.com/api/v1/realtime/ticket');
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer swy_user');
    expect((init.headers as Record<string, string>)['X-Client-Id']).toBe('c1');
  });

  it('never puts the long-lived secret in a query string', async () => {
    const fetchImpl = vi.fn(async () => new Response(JSON.stringify({ ticket: 't' }), { status: 200 }));
    await mintRealtimeTicket(fetchImpl as unknown as typeof fetch, 'https://api.example.com', 'c1', 'swy_secret');
    const [url] = fetchImpl.mock.calls[0] as [string];
    expect(url).not.toMatch(/token=/);
    expect(url).not.toMatch(/swy_secret/);
  });

  it('returns empty on HTTP error so the socket can fall back to open mode', async () => {
    const fetchImpl = vi.fn(async () => new Response('no', { status: 401 }));
    await expect(
      mintRealtimeTicket(fetchImpl as unknown as typeof fetch, '', 'c1', ''),
    ).resolves.toBe('');
  });
});
