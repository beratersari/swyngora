import path from 'node:path';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Proxy target for WSL/dev: browser calls same origin; Vite forwards to Go API.
// Use 127.0.0.1 (not localhost) inside WSL so we hit the Linux stack reliably.
const API_PROXY_TARGET = process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080';

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
    proxy: {
      // Same-origin /api/* and /health → backend (fixes WSL: browser localhost ≠ WSL)
      '/api': {
        target: API_PROXY_TARGET,
        changeOrigin: true,
      },
      '/health': {
        target: API_PROXY_TARGET,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
});
