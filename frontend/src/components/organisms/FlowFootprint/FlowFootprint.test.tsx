import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { FlowFootprint } from './FlowFootprint';

describe('FlowFootprint', () => {
  it('shows buy and sell pressure, the POC, and a strong imbalance', () => {
    renderWithProviders(
      <FlowFootprint
        report={{
          summary: 'Most prints at 65520.',
          combined: {
            truncated: false,
            levels: [
              {
                price: '65520',
                high: '65530',
                buyVolume: '120000',
                sellVolume: '30000',
                volume: '150000',
                delta: '90000',
                sharePct: '40',
                trades: 18,
                isPoc: true,
              },
              {
                price: '65500',
                high: '65510',
                buyVolume: '20000',
                sellVolume: '80000',
                volume: '100000',
                delta: '-60000',
                sharePct: '25',
                trades: 11,
              },
              {
                price: '65480',
                high: '65490',
                buyVolume: '40000',
                sellVolume: '38000',
                volume: '78000',
                delta: '2000',
                sharePct: '20',
                trades: 9,
              },
            ],
          },
        }}
      />,
    );

    expect(screen.getByText(/65[,.]520/)).toBeInTheDocument();
    expect(screen.getAllByText('POC').length).toBeGreaterThan(0);
    expect(screen.getByText(/4\.0× buy/i)).toBeInTheDocument();
    expect(screen.getByText(/4\.0× sell/i)).toBeInTheDocument();
    expect(screen.getByText('+90.00K')).toBeInTheDocument();
    expect(screen.getByText('-60.00K')).toBeInTheDocument();
    const poc = screen.getAllByRole('listitem').find((row) => row.getAttribute('title')?.includes('18 prints'));
    expect(poc?.getAttribute('title')).toContain('buy');
  });

  it('renders nothing when the ladder is empty', () => {
    const { container } = renderWithProviders(<FlowFootprint report={{ summary: 'none', combined: { levels: [] } }} />);
    expect(container).toBeEmptyDOMElement();
  });
});
