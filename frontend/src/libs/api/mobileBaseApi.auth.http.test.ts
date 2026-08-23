import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

/**
 * HTTP e2e of mobile/src/libs/api/baseApi.ts against a server that requires
 * Authorization (same rule as backend API_AUTH_TOKEN).
 */

type Seen = { authorization: string; clientId: string; url: string };
const seen: Seen[] = [];
let baseURL = '';

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
});

afterAll(async () => {
  await new Promise<void>((resolve, reject) => {
    server.close((err) => (err ? reject(err) : resolve()));
  });
});

describe('mobile baseApi against a token-gated watchlist', () => {
  it('must send Authorization or watchlist is 401', async () => {
    vi.resetModules();
    vi.doMock('@/config/env', () => ({
      env: { apiBaseUrl: baseURL, apiBaseUrlLabel: baseURL, runtime: 'web', clientId: '' },
    }));
    vi.doMock('@/libs/utils/clientId', () => ({
      peekClientId: () => 'mobile-e2e',
      getOrCreateClientId: () => 'mobile-e2e',
    }));

    const { baseApi } = await import('../../../../mobile/src/libs/api/baseApi');
    const { watchlistApi } = await import('../../../../mobile/src/libs/api/endpoints/watchlistApi');
    const { configureStore } = await import('@reduxjs/toolkit');

    const store = configureStore({
      reducer: { [baseApi.reducerPath]: baseApi.reducer },
      middleware: (gdm) => gdm().concat(baseApi.middleware),
    });

    const result = await store.dispatch(watchlistApi.endpoints.getWatchlist.initiate());
    const last = seen[seen.length - 1];

    expect(last, `no request recorded; error=${JSON.stringify(result.error)}`).toBeTruthy();
    expect(last.clientId).toBe('mobile-e2e');
    expect(
      last.authorization.startsWith('Bearer '),
      `mobile Authorization=${JSON.stringify(last.authorization)} url=${last.url}`,
    ).toBe(true);
    expect(result.error, JSON.stringify(result.error)).toBeUndefined();
  });
});
