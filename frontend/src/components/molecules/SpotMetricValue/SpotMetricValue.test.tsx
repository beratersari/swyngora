import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SpotMetricValue } from './SpotMetricValue';
import type { SpotMetricDef } from '@/libs/utils';

const priceMetric: SpotMetricDef = {
  id: 'lastPrice',
  field: 'lastPrice',
  format: 'price',
  labelKey: 'last',
  surfaces: ['markets', 'watchlist'],
  defaultVisible: { markets: true, watchlist: true },
};

const changeMetric: SpotMetricDef = {
  id: 'priceChangePercent',
  field: 'priceChangePercent',
  format: 'changePercent',
  labelKey: 'change24h',
  surfaces: ['markets', 'watchlist'],
  defaultVisible: { markets: true, watchlist: true },
  toneFromChange: true,
};

const tagsMetric: SpotMetricDef = {
  id: 'tags',
  field: 'tags',
  format: 'tags',
  labelKey: 'tags',
  surfaces: ['markets'],
  defaultVisible: { markets: true },
};

describe('SpotMetricValue', () => {
  it('formats a price with locale', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={priceMetric}
        spot={{ lastPrice: '1234.5' }}
        locale="en-US"
      />,
    );
    expect(screen.getByText(/1,234\.5/)).toBeInTheDocument();
  });

  it('formats change percent', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={changeMetric}
        spot={{ priceChangePercent: '1.5' }}
      />,
    );
    expect(screen.getByText('+1.50%')).toBeInTheDocument();
  });

  it('renders tags and overflow count', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={tagsMetric}
        spot={{ tags: ['a', 'b', 'c', 'd', 'e'] }}
      />,
    );
    expect(screen.getByText('a')).toBeInTheDocument();
    expect(screen.getByText('+1')).toBeInTheDocument();
  });

  it('shows skeleton when isLoading', () => {
    const { container } = renderWithProviders(
      <SpotMetricValue metric={priceMetric} spot={undefined} isLoading />,
    );
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });

  it('shows dash when tags empty', () => {
    renderWithProviders(<SpotMetricValue metric={tagsMetric} spot={{ tags: [] }} />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('renders delist tag in the tags section when delistTime is set', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={tagsMetric}
        spot={{ tags: ['Meme'], delistTime: '2026-08-17T03:00:00Z' }}
      />,
    );
    expect(screen.getByText(/Delist/i)).toBeInTheDocument();
    expect(screen.getByText(/2026/)).toBeInTheDocument();
    expect(screen.getByText('Meme')).toBeInTheDocument();
  });

  it('labels ETHBTC last with BTC, not venue USDT', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={priceMetric}
        exchange="binance"
        spot={{ symbol: 'ETHBTC', lastPrice: '0.035' }}
        locale="en-US"
      />,
    );
    const text = screen.getByText(/0\.035/).textContent ?? '';
    expect(text).toMatch(/BTC/);
    expect(text).not.toMatch(/USDT/);
  });

  it('renders delist tag alone when product tags are empty', () => {
    renderWithProviders(
      <SpotMetricValue
        metric={tagsMetric}
        spot={{ tags: [], delistTime: '2026-08-17T03:00:00Z' }}
      />,
    );
    expect(screen.getByText(/Delist/i)).toBeInTheDocument();
    expect(screen.queryByText('—')).not.toBeInTheDocument();
  });
});
