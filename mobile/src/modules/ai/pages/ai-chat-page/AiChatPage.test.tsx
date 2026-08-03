import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AiChatPage } from './AiChatPage';
import type { AiChatPageViewModel } from './AiChatPage.types';

const baseVm = (): AiChatPageViewModel => ({
  title: 'Ask',
  draft: '',
  onChangeDraft: vi.fn(),
  onSend: vi.fn(),
  onNewChat: vi.fn(),
  onRetryLast: vi.fn(),
  messages: [],
  isSending: false,
  sendDisabled: true,
  bannerError: null,
  emptyTitle: 'Ask Swyngora',
  emptyMessage: 'Ask about prices, RSI, pumps, or markets.',
  disclaimer: 'Informational only — not financial advice.',
  placeholder: 'Ask about markets…',
  sendLabel: 'Send',
  newChatLabel: 'New chat',
  toolsLabel: 'Tools',
  retryLabel: 'Retry',
  showRetry: false,
});

describe('AiChatPage', () => {
  it('renders empty state and disclaimer', () => {
    renderWithProviders(<AiChatPage viewModel={baseVm()} />);
    expect(screen.getByText('Ask')).toBeTruthy();
    expect(screen.getByText('Ask Swyngora')).toBeTruthy();
    expect(screen.getByText(/informational only/i)).toBeTruthy();
    expect(screen.getByText('New chat')).toBeTruthy();
  });

  it('renders banner error and retry', () => {
    renderWithProviders(
      <AiChatPage
        viewModel={{
          ...baseVm(),
          emptyTitle: '',
          emptyMessage: '',
          bannerError: 'Assistant unavailable',
          showRetry: true,
          messages: [
            { id: '1', role: 'user', text: 'BTC RSI?', error: 'Assistant unavailable' },
          ],
        }}
      />,
    );
    expect(screen.getAllByText('Assistant unavailable').length).toBeGreaterThan(0);
    expect(screen.getByText('Retry')).toBeTruthy();
    expect(screen.getByText('BTC RSI?')).toBeTruthy();
  });

  it('renders assistant message and tools', () => {
    renderWithProviders(
      <AiChatPage
        viewModel={{
          ...baseVm(),
          emptyTitle: '',
          emptyMessage: '',
          messages: [
            { id: 'u', role: 'user', text: 'hi' },
            {
              id: 'a',
              role: 'assistant',
              text: 'Hello from assistant',
              tools: ['market_agent'],
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('Hello from assistant')).toBeTruthy();
    expect(screen.getByText('market_agent')).toBeTruthy();
  });
});
