import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { i18n } from '@/libs/i18n';
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

  const liveRead = {
    symbol: 'BTCUSDT',
    priceCvd: { kind: 'agrees' as const },
    printsMissing: ['binance', 'bybit', 'okx'],
  };
  const entryRead = {
    symbol: 'BTCUSDT',
    priceCvd: { kind: 'price_up_cvd_down' as const },
  };

  async function sendBitcoin(user: ReturnType<typeof userEvent.setup>) {
    await user.type(screen.getByRole('textbox'), 'bitcoin');
    await user.click(screen.getByRole('button', { name: /send|gönder/i }));
  }

  it('shows the same situation sentences as the coin card', async () => {
    const user = userEvent.setup();
    streamChat.mockImplementation(async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
      const final = {
        type: 'final',
        reply: 'Price and futures CVD agree. Measured prints stay blank.',
        reads: [{ symbol: 'BTCUSDT', situation: liveRead }],
      };
      arg.onEvent(final);
      return final;
    });
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await sendBitcoin(user);

    expect(await screen.findByText('Situation · BTCUSDT')).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Situation' })).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(
      screen.getByText('Measured liquidation prints stay blank until Binance, Bybit, and OKX cover the last 24h.'),
    ).toBeInTheDocument();
    expect(screen.queryByText(/mostly long|mostly short/i)).not.toBeInTheDocument();
    expect(screen.queryByText('Entered as')).not.toBeInTheDocument();
  });

  it('shows the open paper buy beside the live read in English and Turkish', async () => {
    const user = userEvent.setup();
    const final = {
      type: 'final',
      reply: 'The entry and the live read.',
      reads: [
        {
          symbol: 'BTCUSDT',
          situation: liveRead,
          entries: [
            {
              exchange: 'binance',
              book: 'Main',
              entryAt: '2026-10-01T00:00:00Z',
              entrySituation: entryRead,
            },
          ],
        },
      ],
    };
    streamChat.mockImplementation(async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
      arg.onEvent(final);
      return final;
    });
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await sendBitcoin(user);

    expect(await screen.findByText('Situation · BTCUSDT')).toBeInTheDocument();
    expect(screen.getByText(/Open paper buy · Main · binance/)).toBeInTheDocument();
    expect(screen.getByRole('group', { name: 'Position read' })).toBeInTheDocument();
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getAllByText('Price and futures CVD are moving together.')).toHaveLength(1);
    expect(
      screen.getByText('Measured liquidation prints stay blank until Binance, Bybit, and OKX cover the last 24h.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Situation' })).not.toBeInTheDocument();

    await i18n.changeLanguage('tr');
    try {
      expect(screen.getByText('Durum · BTCUSDT')).toBeInTheDocument();
      expect(screen.getByText(/Açık kağıt alım · Main · binance/)).toBeInTheDocument();
      expect(screen.getByText('Girişte')).toBeInTheDocument();
      expect(screen.getByText('Fiyat yükselirken vadeli CVD düşüyor.')).toBeInTheDocument();
      expect(screen.getByText('Şimdi')).toBeInTheDocument();
      expect(screen.getByText('Fiyat ile vadeli CVD aynı yönde.')).toBeInTheDocument();
      expect(
        screen.getByText(
          'Ölçülen likidasyon baskıları, Binance, Bybit ve OKX son 24 saati kapatana kadar boş kalır.',
        ),
      ).toBeInTheDocument();
    } finally {
      await i18n.changeLanguage('en');
    }
  });

  it('keeps each open buy and shows the live read once', async () => {
    const user = userEvent.setup();
    streamChat.mockImplementation(async (arg: { onEvent: (ev: Record<string, unknown>) => void }) => {
      const final = {
        type: 'final',
        reply: 'Two books.',
        reads: [
          {
            symbol: 'BTCUSDT',
            situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'agrees' } },
            entries: [
              { exchange: 'binance', entrySituation: entryRead },
              {
                exchange: 'bybit',
                entrySituation: { symbol: 'BTCUSDT', priceCvd: { kind: 'price_down_cvd_up' } },
              },
            ],
          },
        ],
      };
      arg.onEvent(final);
      return final;
    });
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await sendBitcoin(user);

    expect(await screen.findByText(/Open paper buy · binance/)).toBeInTheDocument();
    expect(screen.getByText(/Open paper buy · bybit/)).toBeInTheDocument();
    expect(screen.getAllByText('Entered as')).toHaveLength(2);
    expect(screen.getAllByText('Now')).toHaveLength(1);
    expect(screen.getAllByText('Price and futures CVD are moving together.')).toHaveLength(1);
    expect(screen.getByText('Price is falling while futures CVD is rising.')).toBeInTheDocument();
  });

  it('shows the situation card from the non-stream fallback', async () => {
    const user = userEvent.setup();
    streamChat.mockRejectedValue({ status: 404 });
    postChat.mockResolvedValue({
      reply: 'Fallback read.',
      sessionId: 's',
      reads: [{ symbol: 'BTCUSDT', situation: liveRead }],
    });
    renderWithProviders(<AiChatPage />, { routerEntries: ['/ai'] });
    await sendBitcoin(user);
    expect(await screen.findByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(
      screen.getByText('Measured liquidation prints stay blank until Binance, Bybit, and OKX cover the last 24h.'),
    ).toBeInTheDocument();
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
