import path from 'node:path';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Proxy target for WSL/dev: browser calls same origin; Vite forwards to Go API.
// Use 127.0.0.1 (not localhost) inside WSL so we hit the Linux stack reliably.
const API_PROXY_TARGET = process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080';

/**
 * Optional: inject API auth for local dev when backend has API_AUTH_TOKEN set.
 * Read from process env only (Vite server) — never bundled into client JS.
 * Prefer VITE_DEV_API_AUTH_TOKEN or API_AUTH_TOKEN in the shell that runs `npm run dev`.
 */
const DEV_API_AUTH_TOKEN = (
  process.env.VITE_DEV_API_AUTH_TOKEN ||
  process.env.API_AUTH_TOKEN ||
  ''
).trim();

function injectDevApiAuth(
  proxyReq: { setHeader: (name: string, value: string) => void },
): void {
  if (!DEV_API_AUTH_TOKEN) return;
  proxyReq.setHeader('Authorization', `Bearer ${DEV_API_AUTH_TOKEN}`);
}

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    host: '0.0.0.0', // reachable from Windows host / LAN (WSL)
    port: 5174,
    strictPort: true,
    // /mnt/c (Windows FS) does not reliably emit inotify events; poll so HMR works.
    watch: {
      usePolling: true,
      interval: 300,
    },
    proxy: {
      // Same-origin /api/* and /health → backend (fixes WSL: browser localhost ≠ WSL)
      '/api': {
        target: API_PROXY_TARGET,
        changeOrigin: true,
        ws: true,
        // AI multi-agent turns can exceed default proxy idle timeouts.
        timeout: 360_000,
        proxyTimeout: 360_000,
        configure: (proxy) => {
          proxy.on('proxyReq', injectDevApiAuth);
          proxy.on('proxyReqWs', injectDevApiAuth);
        },
      },
      '/health': {
        target: API_PROXY_TARGET,
        changeOrigin: true,
      },
    },
  },
  test: {
    exclude: ['**/node_modules/**', '**/dist/**', '**/e2e/**', '**/*.e2e.*'],
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'html', 'json-summary'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.types.ts',
        'src/**/*.styles.ts',
        'src/**/index.ts',
        'src/libs/api/generated/**',
        'src/test/**',
        'src/vite-env.d.ts',
        'src/main.tsx',
        'src/styles/styled.d.ts',
      ],
    },
  },
});
