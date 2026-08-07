import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AlertsPage } from './AlertsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListPriceAlertsQuery: () => ({
      data: { alerts: [] },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useCreatePriceAlertMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useDeletePriceAlertMutation: () => [vi.fn(), { isLoading: false }],
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

describe('AlertsPage', () => {
  it('renders title and create form', async () => {
    renderWithProviders(<AlertsPage />, { routerEntries: ['/alerts'] });
    expect(await screen.findByRole('button', { name: /create|oluştur/i })).toBeInTheDocument();
  });
});
