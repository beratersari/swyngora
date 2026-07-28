import { createRoot } from 'react-dom/client';
import { enableScreens } from 'react-native-screens';
import { initI18n } from '@/libs/i18n';
import { App } from '@/app/App';

// Localize before first paint so tab labels and screens share one language.
initI18n();

// react-native-screens often renders blank scenes on react-native-web.
// Disable so tab/stack content fills the viewport in Chrome.
enableScreens(false);

const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('Root element #root not found');
}

createRoot(rootEl).render(<App />);
