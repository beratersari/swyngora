import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { TickerTape } from './TickerTape';

describe('TickerTape', () => {
  it('renders nothing without items', () => {
    const { container } = renderWithProviders(<TickerTape items={[]} ariaLabel="Tape" />);
    expect(container.querySelector('[role="region"]')).toBeNull();
  });

  it('links each symbol into coin detail', () => {
    const { getAllByRole } = renderWithProviders(
      <TickerTape
        ariaLabel="Top movers"
        items={[
          {
            exchange: 'binance',
            symbol: 'ETHUSDT',
            lastPrice: '3,200',
            changePercent: '+2.00%',
            changeValue: 2,
            href: '/markets/binance/ETHUSDT',
          },
        ]}
      />,
    );
    const links = getAllByRole('link', { name: /ETH\/USDT/ });
    expect(links.length).toBeGreaterThanOrEqual(1);
    expect(links[0]).toHaveAttribute('href', '/markets/binance/ETHUSDT');
  });
});
