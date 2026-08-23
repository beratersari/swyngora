import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { getBrowserApiToken, setBrowserApiToken } from '@/libs/utils/apiAuth';
import { SettingsPage } from './SettingsPage';

const READ_SECRET = 'swy_readlock000000000000000000000000000000';
const TRADE_SECRET = 'swy_tradelock00000000000000000000000000000';

const createKey = vi.fn();

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
    useCreateAccountAPIKeyMutation: () => [createKey, { isLoading: false }],
    useRevokeAccountAPIKeyMutation: () => [vi.fn()],
    useListExportsQuery: () => empty,
    useStartExportMutation: () => [vi.fn(), { isLoading: false }],
    useCancelExportMutation: () => [vi.fn()],
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

async function createNamedKey() {
  const user = userEvent.setup();
  renderWithProviders(<SettingsPage />, { routerEntries: ['/settings'] });
  const name = screen.getByLabelText(/^name$/i);
  await user.type(name, 'desk-read');
  await user.click(screen.getByRole('button', { name: /create key/i }));
  await waitFor(() => {
    expect(createKey).toHaveBeenCalled();
  });
}

describe('SettingsPage API key session binding', () => {
  beforeEach(() => {
    setBrowserApiToken('');
    createKey.mockReset();
    createKey.mockImplementation(() => ({
      unwrap: async () => ({ secret: READ_SECRET, permission: 'read' }),
    }));
  });

  it('does not install a newly created read-only key as the browser session token', async () => {
    await createNamedKey();
    expect(createKey.mock.calls[0]?.[0]).toMatchObject({ permission: 'read' });
    // Read keys 403 every POST (orders, cash, watchlist, even creating a trade key).
    expect(getBrowserApiToken()).not.toBe(READ_SECRET);
  });

  it('does not replace an existing trade session token with a new read-only key', async () => {
    setBrowserApiToken(TRADE_SECRET);
    await createNamedKey();
    expect(getBrowserApiToken()).toBe(TRADE_SECRET);
  });
});
