import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { AiChatPage } from './AiChatPage';

const postChat = vi.fn();
const streamChat = vi.fn();

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    streamAiChat: (...args: unknown[]) => streamChat(...args),
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
    streamChat.mockReset();
    localStorage.clear();
  });

  it('renders title and disclaimer', () => {
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    expect(screen.getByText(/AI assistant|AI asistan/i)).toBeInTheDocument();
    expect(screen.getByText(/not financial advice|yatırım tavsiyesi/i)).toBeInTheDocument();
  });

  it('sends a message and shows the reply', async () => {
    const user = userEvent.setup();
    streamChat.mockImplementation(async (arg: { onEvent: (ev: { type: string; text?: string; reply?: string; tools?: string[]; thinking?: string[] }) => void }) => {
      arg.onEvent({ type: 'thinking', text: 'check RSI' });
      arg.onEvent({
        type: 'final',
        reply: 'BTC looks interesting on the 1h.',
        tools: ['market_agent'],
        thinking: ['check RSI'],
      });
      return { type: 'final', reply: 'BTC looks interesting on the 1h.', tools: ['market_agent'] };
    });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });

    const box = screen.getByRole('textbox');
    await user.type(box, 'What is BTC RSI?');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    await waitFor(() => {
      expect(streamChat).toHaveBeenCalled();
    });
    expect(await screen.findByText('What is BTC RSI?')).toBeInTheDocument();
    expect(await screen.findByText('BTC looks interesting on the 1h.')).toBeInTheDocument();
    expect(screen.getByText(/Thinking · 1 step|Düşünme · 1 adım/)).toBeInTheDocument();
    expect(screen.queryByText(/^Think$|^Düşünce$/)).not.toBeInTheDocument();
    await user.click(screen.getByText(/Thinking · 1 step|Düşünme · 1 adım/));
    expect(screen.getByText(/^Think$|^Düşünce$/)).toBeInTheDocument();
    expect(screen.getByText('check RSI')).toBeInTheDocument();
    expect(screen.getByText('market_agent')).toBeInTheDocument();
    expect(screen.queryByText(/get_indicators →/)).not.toBeInTheDocument();
  });

  it('renders markdown reply and numbered process steps', async () => {
    const user = userEvent.setup();
    streamChat.mockImplementation(async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
      arg.onEvent({ type: 'status', text: 'Planning…' });
      arg.onEvent({ type: 'status', text: 'Orchestrator running…' });
      arg.onEvent({ type: 'thinking', text: 'Running market_agent…' });
      arg.onEvent({
        type: 'final',
        reply: '**BTCUSDT 1h RSI (14):** 59.32\n\n- Neutral zone',
        tools: [
          'market_agent(task=Get current BTCUSDT 1h RSI)',
          '↳ get_indicators → { "latest": { "rsi": 59.32 } }',
        ],
      });
      return { type: 'final', reply: '**BTCUSDT 1h RSI (14):** 59.32\n\n- Neutral zone' };
    });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'rsi?');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    expect(await screen.findByText('59.32')).toBeInTheDocument();
    expect(screen.getByText('Neutral zone')).toBeInTheDocument();
    expect(screen.getByText(/Thinking · 3 steps|Düşünme · 3 adım/)).toBeInTheDocument();
    expect(screen.queryByText('Planning…')).not.toBeInTheDocument();
    await user.click(screen.getByText(/Thinking · 3 steps|Düşünme · 3 adım/));
    expect(screen.getByText('Planning…')).toBeInTheDocument();
    expect(screen.getByText('Running market_agent…')).toBeInTheDocument();
    expect(screen.queryByText(/"latest"/)).not.toBeInTheDocument();
    expect(screen.getByText('market_agent')).toBeInTheDocument();
    expect(screen.getByText('get_indicators')).toBeInTheDocument();
  });

  it('shows the process panel while the stream is still open', async () => {
    const user = userEvent.setup();
    let release!: (ev: { type: string; reply: string }) => void;
    const gate = new Promise<{ type: string; reply: string }>((resolve) => {
      release = resolve;
    });
    streamChat.mockImplementation(
      async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
        arg.onEvent({ type: 'status', text: 'Planning…' });
        arg.onEvent({ type: 'thinking', text: 'Need ticker then RSI' });
        arg.onEvent({ type: 'tool', text: 'market_agent(task=BTC RSI)' });
        const final = await gate;
        arg.onEvent(final);
        return final;
      },
    );

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'rsi?');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    expect(await screen.findByText(/Process|Süreç/)).toBeInTheDocument();
    expect(screen.getByText(/Thinking|Düşünme/)).toBeInTheDocument();
    expect(screen.getByText('Need ticker then RSI')).toBeInTheDocument();
    expect(screen.getByText('market_agent(task=BTC RSI)')).toBeInTheDocument();
    expect(screen.getByText(/Composing answer|Yanıt yazılıyor/)).toBeInTheDocument();

    release({ type: 'final', reply: 'RSI 55' });
    expect(await screen.findByText('RSI 55')).toBeInTheDocument();
    expect(screen.queryByText('Need ticker then RSI')).not.toBeInTheDocument();
    await user.click(screen.getByText(/Thinking · 3 steps|Düşünme · 3 adım/));
    expect(screen.getByText('Need ticker then RSI')).toBeInTheDocument();
  });

  it('shows clickable source URLs from research', async () => {
    const user = userEvent.setup();
    streamChat.mockImplementation(async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
      arg.onEvent({ type: 'thinking', text: 'research' });
      arg.onEvent({
        type: 'final',
        reply: 'BTC near 65k. Not financial advice.',
        tools: ['web_agent'],
        references: [
          {
            title: 'Bitcoin price',
            url: 'https://coinmarketcap.com/currencies/bitcoin/',
            source: 'web',
          },
        ],
      });
      return { type: 'final', reply: 'BTC near 65k. Not financial advice.' };
    });
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'btc news');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));
    const link = await screen.findByRole('link', { name: /bitcoin price/i });
    expect(link).toHaveAttribute('href', 'https://coinmarketcap.com/currencies/bitcoin/');
  });

  it('shows error bubble when the API fails', async () => {
    const user = userEvent.setup();
    streamChat.mockRejectedValue({ status: 502 });

    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await user.type(screen.getByRole('textbox'), 'hello');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));

    expect(await screen.findByText('mapped error')).toBeInTheDocument();
  });
});
