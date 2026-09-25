import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { OptionsStrikeOI } from './OptionsStrikeOI';

const expiry = '2026-03-27T08:00:00Z';

describe('OptionsStrikeOI', () => {
  it('shows call and put open-interest change for the selected expiry', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <OptionsStrikeOI
        expiry={expiry}
        windows={[
          {
            window: '24h',
            complete: true,
            strikes: [
              {
                expiry,
                strike: '100000',
                callOi: '30',
                putOi: '8',
                callPast: '10',
                putPast: '8',
                callChange: '20',
                putChange: '0',
                netChange: '20',
              },
              {
                expiry,
                strike: '90000',
                callOi: '5',
                putOi: '12',
                callPast: '2',
                putPast: '20',
                callChange: '3',
                putChange: '-8',
                netChange: '-5',
              },
            ],
          },
          { window: '1h', complete: false, strikes: [] },
        ]}
      />,
    );
    expect(screen.getByText(/100[.,]000/)).toBeInTheDocument();
    expect(screen.getByText(/calls \+20/i)).toBeInTheDocument();
    expect(screen.getByText(/puts -8/i)).toBeInTheDocument();

    await user.click(screen.getByText('1 hour'));
    expect(screen.getByText(/starts filling in/i)).toBeInTheDocument();
  });
});
