import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { RebalanceResultsTable } from './RebalanceResultsTable';

describe('RebalanceResultsTable', () => {
  it('shows fill counts and leftover drift', () => {
    renderWithProviders(
      <RebalanceResultsTable
        onExpand={vi.fn()}
        results={[
          {
            id: 'r1',
            basketName: 'Core 50/30/20',
            status: 'partial',
            plannedLegs: 5,
            filledLegs: 4,
            failedLegs: 1,
            remainingAbsDriftPct: 4.2,
            totalFee: 1.3,
            totalSlippage: 2.6,
            totalCost: 3.9,
            createdAt: '2026-09-13T12:00:00Z',
            summary: 'Filled 4 of 5 planned legs; 1 failed.',
          },
        ]}
      />,
    );
    expect(screen.getByText(/Core 50\/30\/20/)).toBeInTheDocument();
    expect(screen.getByText('4/5')).toBeInTheDocument();
    expect(screen.getByText('4.20%')).toBeInTheDocument();
  });
});
