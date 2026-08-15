import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CurrencySwitcher } from './CurrencySwitcher';

describe('CurrencySwitcher', () => {
  it('renders currency combobox', () => {
    renderWithProviders(<CurrencySwitcher />);
    expect(screen.getByRole('combobox', { name: /currency|para birimi/i })).toBeInTheDocument();
  });
});
