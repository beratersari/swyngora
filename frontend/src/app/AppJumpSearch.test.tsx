import { describe, expect, it } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AppJumpSearch } from './AppJumpSearch';

describe('AppJumpSearch', () => {
  it('renders a jump combobox on a venue route', () => {
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<AppJumpSearch />} />
      </Routes>,
      { routerEntries: ['/markets/bybit/ETHUSDT'] },
    );
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });
});
