import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

/**
 * E2E: the mobile RTK client talks to a real HTTP server that requires
 * Authorization (same gate as backend APIAuth when API_AUTH_TOKEN is set).
 * Web attaches the browser token; mobile prepareHeaders only sets X-Client-Id.
 */

type Seen = { authorization: string; clientId: string; url: string };

let baseURL = '';
const seen: Seen[] = [];

const server = createServer((req: IncomingMessage, res: ServerResponse) => {
  const url = req.url ?? '';
  seen.push({
    authorization: String(req.headers.authorization ?? ''),
    clientId: String(req.headers['x-client-id'] ?? ''),
    url,
  });
  if (url.startsWith('/api/v1/watchlist') && !req.headers.authorization) {
    res.statusCode = 401;
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({ error: { code: 'unauthorized', message: 'missing token' } }));
    return;
  }
  res.statusCode = 200;
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify({ clientId: 'ok', items: [], count: 0 }));
});

beforeAll(async () => {
  await new Promise<void>((resolve) => {
    server.listen(0, '127.0.0.1', () => resolve());
  });
  const addr = server.address();
  if (!addr || typeof addr === 'string') {
    throw new Error('server did not bind');
  }
  baseURL = `http://127.0.0.1:${addr.port}`;
  vi.resetModules();
  vi.doMock('@/config/env', () => ({
    env: { apiBaseUrl: baseURL, apiBaseUrlLabel: baseURL, runtime: 'web' },
  }));
});

afterAll(async () => {
  await new Promise<void>((resolve, reject) => {
    server.close((err) => (err ? reject(err) : resolve()));
  });
});

describe('mobile RTK client against a token-gated API', () => {
  it('sends Authorization so watchlist does not 401 when API auth is on', async () => {
    const { store } = await import('./store');
    const { watchlistApi } = await import('./endpoints/watchlistApi');

    const result = await store.dispatch(watchlistApi.endpoints.getWatchlist.initiate());
    const last = seen[seen.length - 1];

    expect(last, 'mobile must issue a watchlist request').toBeTruthy();
    expect(last.clientId, 'X-Client-Id is set').toBeTruthy();
    expect(
      last.authorization.startsWith('Bearer '),
      `mobile request Authorization=${JSON.stringify(last.authorization)} url=${last.url}`,
    ).toBe(true);
    expect(result.error, `watchlist failed: ${JSON.stringify(result.error)}`).toBeUndefined();
    expect(result.data).toMatchObject({ items: [] });
  });
});
