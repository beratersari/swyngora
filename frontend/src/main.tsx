import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@/libs/i18n';
import { App } from '@/app/App';

const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('Root element #root not found');
}

createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
