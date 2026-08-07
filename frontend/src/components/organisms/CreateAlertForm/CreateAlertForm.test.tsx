import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CreateAlertForm } from './CreateAlertForm';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

describe('CreateAlertForm', () => {
  it('renders create controls', () => {
    renderWithProviders(
      <CreateAlertForm defaultSymbol="BTCUSDT" onSubmit={async () => undefined} />,
    );
    expect(screen.getByRole('button', { name: /create|oluştur/i })).toBeInTheDocument();
  });
});
