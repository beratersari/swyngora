import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SignalsRulesTable } from './SignalsRulesTable';

describe('SignalsRulesTable', () => {
  it('renders a rule row', () => {
    renderWithProviders(
      <SignalsRulesTable
        items={[
          {
            id: 'r1',
            type: 'rsi',
            interval: '4h',
            enabled: true,
            rsiPeriod: 14,
            rsiCondition: 'below',
            rsiThreshold: 40,
          },
        ]}
        onDelete={() => undefined}
        onToggle={() => undefined}
        onEdit={() => undefined}
      />,
    );
    expect(screen.getByText(/RSI\(14\)/)).toBeInTheDocument();
  });
});
