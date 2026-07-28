import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { MarketsToolbar } from './MarketsToolbar';

describe('MarketsToolbar', () => {
  const base = {
    q: '',
    quote: 'USDT',
    tag: '',
    tags: ['Meme', 'DeFi'],
    tagsLoading: false,
    onQChange: vi.fn(),
    onQuoteChange: vi.fn(),
    onTagChange: vi.fn(),
  };

  it('renders search with accessible name', () => {
    renderWithTheme(<MarketsToolbar {...base} />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('calls onQChange when typing', async () => {
    const user = userEvent.setup();
    const onQChange = vi.fn();
    renderWithTheme(<MarketsToolbar {...base} onQChange={onQChange} />);
    await user.type(screen.getByRole('textbox'), 'btc');
    expect(onQChange).toHaveBeenCalled();
  });
});
