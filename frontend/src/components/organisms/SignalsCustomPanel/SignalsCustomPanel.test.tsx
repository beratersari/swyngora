import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { defaultCustomWeightDraft } from './helpers';
import { SignalsCustomPanel } from './SignalsCustomPanel';

describe('SignalsCustomPanel', () => {
  it('shows the mixer and live preview score', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    renderWithProviders(
      <SignalsCustomPanel
        profiles={[]}
        selectedId=""
        draft={defaultCustomWeightDraft().map((row) =>
          row.id === 'ema_crossover'
            ? { ...row, enabled: true, pct: 100 }
            : { ...row, enabled: false, pct: 0 },
        )}
        items={[
          {
            exchange: 'binance',
            symbol: 'ETHUSDT',
            custom: {
              score: 62,
              grade: 'B',
              evidence: { score: 40, label: 'weak', detail: 'Evidence is weak (40) — 1 fresh, 0 old, 0 missing; Used 3 of 5 setups', fresh: 1, old: 0, missing: 0 },
              factors: [{ id: 'ema_crossover', score: 90, status: 'used', requestedPct: 16, sharePct: 16, effect: 6.4, quality: 100, freshness: 'fresh' }],
            },
            scoreCompare: { default: { score: 56 }, applied: { score: 62 }, delta: { score: 6 } },
            tapeSetups: 5,
            tapeUsed: 3,
            tapeThin: 2,
            tapeFresh: { note: 'oi 6m ago; takers 1m ago; book 2m ago; options 40m ago (old)' },
            engine: {
              side: 'long',
              levels: { entry: 2400, stopLoss: 2320, takeProfit: 2580, leverage: 4 },
            },
          },
        ]}
        onSelectProfile={vi.fn()}
        onDraft={vi.fn()}
        onSave={vi.fn()}
        onSaveAs={vi.fn()}
        onDuplicate={vi.fn()}
        onSetDefault={vi.fn()}
        onDelete={vi.fn()}
        onApply={onApply}
        onReset={vi.fn()}
        onOpen={vi.fn()}
      />,
    );
    expect(screen.getByTestId('signals-custom-total')).toBeInTheDocument();
    expect(screen.getByText(/ETH/i)).toBeInTheDocument();
    expect(screen.getByTestId('signals-custom-tape-stats')).toHaveTextContent(
      /Used 3 of 5 similar setups; 2 had missing tape|Kullanılan 3 \/ 5 benzer kurulum; 2 tanesinde eksik bant vardı/,
    );
    expect(screen.getByTestId('signals-custom-tape-fresh')).toHaveTextContent(/oi 6m ago/);
    expect(screen.getByTestId('signals-custom-evidence')).toBeInTheDocument();
    expect(screen.getByTestId('signals-custom-evidence-counts')).toHaveTextContent(
      /1 fresh, 0 old, 0 missing|1 taze, 0 eski, 0 eksik/,
    );
    expect(screen.getByTestId('signals-custom-levels')).toHaveTextContent(/4x/);
    await user.click(screen.getByRole('button', { name: /apply mix|karışımı uygula/i }));
    expect(onApply).toHaveBeenCalled();
  });
});
