import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { AiChatPage } from './AiChatPage';

const postChat = vi.fn();

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    usePostAiChatMutation: () => [
      (arg: unknown) => ({
        unwrap: () => postChat(arg),
      }),
      { isLoading: false },
    ],
    rtkErrorMessage: () => 'mapped error',
  };
});

describe('AiChatPage', () => {
  beforeEach(() => {
    postChat.mockReset();
    localStorage.clear();
  });

  it('renders title and disclaimer', () => {
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    expect(screen.getByText(/AI assistant|AI asistan/i)).toBeInTheDocument();
    expect(screen.getByText(/not financial advice|yatırım tavsiyesi/i)).toBeInTheDocument();
  });

  it('sends a message and shows the reply', async () => {
    const user = userEvent.setup();
    postChat.mockResolvedValue({
      reply: 'BTC looks interesting on the 1h.',
      sessionId: 'web-ai-test',
      tools: ['market_agent'],
      thinking: ['check RSI'],
      note: 'Informational only — not financial advice.',
    });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });

    const box = screen.getByRole('textbox');
    await user.type(box, 'What is BTC RSI?');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    await waitFor(() => {
      expect(postChat).toHaveBeenCalled();
    });
    expect(await screen.findByText('What is BTC RSI?')).toBeInTheDocument();
    expect(await screen.findByText('BTC looks interesting on the 1h.')).toBeInTheDocument();
    expect(screen.getByText('market_agent')).toBeInTheDocument();
    expect(screen.queryByText(/get_indicators →/)).not.toBeInTheDocument();
  });

  it('renders markdown reply and collapses long traces', async () => {
    const user = userEvent.setup();
    postChat.mockResolvedValue({
      reply: '**BTCUSDT 1h RSI (14):** 59.32\n\n- Neutral zone',
      sessionId: 'web-ai-test',
      tools: [
        'market_agent(task=Get current BTCUSDT 1h RSI)',
        '↳ get_indicators → { "latest": { "rsi": 59.32 } }',
      ],
      thinking: ['Planning…', 'Orchestrator running…', 'Running market_agent…'],
    });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'rsi?');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    expect(await screen.findByText('59.32')).toBeInTheDocument();
    expect(screen.getByText('Neutral zone')).toBeInTheDocument();
    expect(screen.getByText(/Thinking · 3/i)).toBeInTheDocument();
    expect(screen.queryByText(/"latest"/)).not.toBeInTheDocument();
    expect(screen.getByText('market_agent')).toBeInTheDocument();
    expect(screen.getByText('get_indicators')).toBeInTheDocument();
  });

  it('shows error bubble when the API fails', async () => {
    const user = userEvent.setup();
    postChat.mockRejectedValue({ status: 502 });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'hello');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    expect(await screen.findByText('mapped error')).toBeInTheDocument();
  });
});
