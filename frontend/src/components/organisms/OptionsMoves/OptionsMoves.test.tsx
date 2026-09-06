import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsMoves } from './OptionsMoves';

describe('OptionsMoves', () => {
  it('shows an important options movement', () => {
    renderWithProviders(
      <OptionsMoves
        data={{
          moves: {
            findings: [
              {
                kind: 'oi_jump',
                topic: 'oi',
                window: '1h',
                severity: 'notable',
                title: 'Open interest jumped 14% in 1h — well above typical',
                detail: 'Open interest moved up 14.0% over 1h.',
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText(/open interest jumped 14%/i)).toBeInTheDocument();
    expect(screen.getByText(/1 hour/i)).toBeInTheDocument();
  });

  it('renders nothing when there is no unusual move', () => {
    const { container } = renderWithProviders(
      <OptionsMoves data={{ moves: { findings: [] } }} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
