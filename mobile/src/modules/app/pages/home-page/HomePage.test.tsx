import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { HomePage } from './HomePage';
import type { HomePageViewModel } from './HomePage.types';

const stubVm: HomePageViewModel = {
  title: 'Swyngora',
  apiBaseUrlLabel: '(same origin / Vite proxy)',
  healthStatus: 'ok',
  healthDetail: 'up',
  isLoading: false,
  isPollingPaused: false,
  errorMessage: null,
  onRetry: () => undefined,
  onOpenMarkets: () => undefined,
  onOpenPumps: () => undefined,
  onOpenAsk: () => undefined,
};

describe('HomePage', () => {
  it('renders injected view model', () => {
    render(<HomePage viewModel={stubVm} />);
    expect(screen.getByText('Swyngora')).toBeTruthy();
    expect(screen.getByText(/OK/)).toBeTruthy();
  });
});
