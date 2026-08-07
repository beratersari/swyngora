import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { BrandTag } from './BrandTag';

describe('BrandTag', () => {
  it('renders children', () => {
    renderWithTheme(<BrandTag variant="live">updating…</BrandTag>);
    expect(screen.getByText('updating…')).toBeInTheDocument();
  });

  it('renders exchange chips', () => {
    renderWithTheme(<BrandTag variant="exchange">binance</BrandTag>);
    expect(screen.getByText('binance')).toBeInTheDocument();
  });
});
