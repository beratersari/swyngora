import { createRoot } from 'react-dom/client';
import { enableScreens } from 'react-native-screens';
import { App } from '@/app/App';

// react-native-screens often renders blank scenes on react-native-web.
// Disable so tab/stack content fills the viewport in Chrome.
enableScreens(false);

const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('Root element #root not found');
}

createRoot(rootEl).render(<App />);
