import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { HolderPanel } from './HolderPanel';

describe('HolderPanel', () => {
  it('renders holder stats and top wallets', () => {
    renderWithTheme(
      <HolderPanel
        holders={{
          asset: 'BTC',
          name: 'Bitcoin',
          holderCount: 50_708_169,
          dailyActive: 963_625,
          topTenSharePct: 5.4,
          topFiftySharePct: 10.86,
          topHundredSharePct: 13.72,
          topHolders: [
            {
              address: '34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo',
              label: 'Binance',
              balance: 248597,
              sharePct: 1.18,
              resolvedBalance: 248597,
              usdValue: 14_915_820_000,
            },
          ],
          source: 'coinmarketcap',
        }}
      />,
    );
    expect(screen.getByText('Holders')).toBeInTheDocument();
    expect(screen.getByText('Addresses')).toBeInTheDocument();
    expect(screen.getByText('50.71M')).toBeInTheDocument();
    expect(screen.getByText('5.4%')).toBeInTheDocument();
    expect(screen.getByText('Binance')).toBeInTheDocument();
    expect(screen.getByText('34xp4v…wseo')).toBeInTheDocument();
    expect(screen.getByText(/248/)).toBeInTheDocument();
  });

  it('shows a soft error without dash stats', () => {
    renderWithTheme(<HolderPanel error="not published" />);
    expect(screen.getByText('not published')).toBeInTheDocument();
    expect(screen.queryByText('Addresses')).not.toBeInTheDocument();
  });

  it('explains a published count with no wallet rows', () => {
    renderWithTheme(
      <HolderPanel
        holders={{
          asset: 'FOO',
          holderCount: 12,
          topHolders: [],
          source: 'coinmarketcap',
        }}
      />,
    );
    expect(screen.getByText(/no top wallets were listed/i)).toBeInTheDocument();
  });
});
