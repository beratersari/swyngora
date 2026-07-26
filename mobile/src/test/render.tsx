import type { ReactElement } from 'react';
import { render } from '@testing-library/react';
import { Provider } from 'react-redux';
import { store } from '@/libs/api';

export function renderWithProviders(ui: ReactElement) {
  return render(<Provider store={store}>{ui}</Provider>);
}
