import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsVsMarket } from './OptionsVsMarket';

describe('OptionsVsMarket', () => {
  it('shows only real disagreements', () => {
    renderWithProviders(
      <OptionsVsMarket
        data={{
          vsMarket: {
            compared: true,
            findings: [
              {
                kind: 'puts_vs_futures_long',
                topic: 'lean',
                severity: 'notable',
                title: 'Options lean put while futures accounts are long',
                detail: 'Put/call OI 2.00 vs 64% long accounts.',
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText(/lean put while futures accounts are long/i)).toBeInTheDocument();
    expect(screen.getByText(/put\/call oi 2/i)).toBeInTheDocument();
  });

  it('renders nothing when options agree with the tape', () => {
    const { container } = renderWithProviders(
      <OptionsVsMarket data={{ vsMarket: { compared: true, findings: [] } }} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
