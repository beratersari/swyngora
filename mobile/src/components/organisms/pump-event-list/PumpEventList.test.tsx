import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PumpEventList } from './PumpEventList';

describe('PumpEventList', () => {
  it('renders events and disclaimer', () => {
    render(
      <PumpEventList
        rows={[
          {
            id: '1',
            returnLabel: '+9.00%',
            returnTone: 'success',
            timeLabel: 'Jan 1',
            metaLabel: 'Close return',
          },
        ]}
        disclaimer="Informational only"
      />,
    );
    expect(screen.getByText('+9.00%')).toBeTruthy();
    expect(screen.getByText('Informational only')).toBeTruthy();
  });
});
