import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App } from './App';

vi.mock('./routes', () => ({
  AppRoutes: () => <div data-testid="routes" />,
}));

describe('App shell navigation', () => {
  it('exposes navigation landmark and markets links', () => {
    render(<App />);
    expect(screen.getByRole('navigation')).toBeInTheDocument();
    const links = screen.getAllByRole('link');
    expect(links.some((a) => a.getAttribute('href') === '/markets')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/watchlist')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/signals')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/pumps')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/ai')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/alerts')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/compare')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/heatmap')).toBe(true);
    expect(links.some((a) => a.getAttribute('href') === '/settings')).toBe(true);
    expect(screen.getByTestId('routes')).toBeInTheDocument();
    expect(screen.getByRole('region', { name: /market prices|piyasa fiyat/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /binance/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /coinbase/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /bist/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /watchlist|izleme/i })).toBeInTheDocument();
  });
});
