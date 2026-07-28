import path from 'node:path';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const API_PROXY_TARGET = process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080';

const rnExtensions = [
  '.web.tsx',
  '.web.ts',
  '.tsx',
  '.ts',
  '.web.jsx',
  '.web.js',
  '.jsx',
  '.js',
  '.json',
];

export default defineConfig({
  plugins: [react()],
  resolve: {
    extensions: rnExtensions,
    alias: {
      '@': path.resolve(__dirname, 'src'),
      'react-native': 'react-native-web',
    },
  },
  define: {
    global: 'globalThis',
    __DEV__: JSON.stringify(true),
  },
  optimizeDeps: {
    esbuildOptions: {
      mainFields: ['module', 'main'],
      resolveExtensions: rnExtensions,
    },
    include: [
      'react-native-web',
      'react-native-gesture-handler',
      'react-native-safe-area-context',
      'react-native-screens',
      'react-native-svg',
      'lucide-react-native',
      '@react-navigation/native',
      '@react-navigation/native-stack',
      '@react-navigation/bottom-tabs',
    ],
  },
  server: {
    host: '0.0.0.0',
    port: 5180,
    strictPort: true,
    proxy: {
      // AI chat can take minutes on local Ollama (multi-agent + tools).
      '/api': {
        target: API_PROXY_TARGET,
        changeOrigin: true,
        timeout: 600_000,
        proxyTimeout: 600_000,
      },
      '/health': { target: API_PROXY_TARGET, changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    server: {
      deps: {
        inline: [
          'react-native',
          'react-native-web',
          'react-native-safe-area-context',
          'react-native-gesture-handler',
          'react-native-screens',
          'react-native-svg',
          'lucide-react-native',
          '@react-navigation/native',
          '@react-navigation/native-stack',
          '@react-navigation/bottom-tabs',
        ],
      },
    },
  },
});
