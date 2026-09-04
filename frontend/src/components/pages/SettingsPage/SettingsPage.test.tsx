import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SettingsPage } from './SettingsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  const empty = {
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  };
  return {
    ...actual,
    useListAccountAPIKeysQuery: () => empty,
    useCreateAccountAPIKeyMutation: () => [vi.fn(), { isLoading: false }],
    useRevokeAccountAPIKeyMutation: () => [vi.fn()],
    useListExportsQuery: () => empty,
    useStartExportMutation: () => [vi.fn(), { isLoading: false }],
    useCancelExportMutation: () => [vi.fn()],
    useListImportsQuery: () => empty,
    usePreviewImportMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useConfirmImportMutation: () => [vi.fn(), { isLoading: false }],
    useCancelImportMutation: () => [vi.fn()],
    useListWatchlistSharesQuery: () => empty,
    useShareWatchlistMutation: () => [vi.fn()],
    useRevokeWatchlistShareMutation: () => [vi.fn()],
    useListPortfolioSharesQuery: () => empty,
    useSharePortfolioMutation: () => [vi.fn()],
    useRevokePortfolioShareMutation: () => [vi.fn()],
    useListRecurringBuysQuery: () => empty,
    useCreateRecurringBuyMutation: () => [vi.fn(), { isLoading: false }],
    usePauseRecurringBuyMutation: () => [vi.fn()],
    useResumeRecurringBuyMutation: () => [vi.fn()],
    useDeleteRecurringBuyMutation: () => [vi.fn()],
  };
});

describe('SettingsPage', () => {
  it('renders settings title and tabs', () => {
    renderWithProviders(<SettingsPage />);
    expect(screen.getByRole('heading', { name: /settings/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /api keys/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /export/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /import/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /sharing/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /recurring/i })).toBeInTheDocument();
  });

  it('shows import preview controls on the Import tab', async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.click(screen.getByRole('tab', { name: /import/i }));
    expect(screen.getByTestId('import-file')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /preview/i })).toBeDisabled();
    expect(screen.getByText(/which custom signal mixes will be added/i)).toBeInTheDocument();
  });
});
