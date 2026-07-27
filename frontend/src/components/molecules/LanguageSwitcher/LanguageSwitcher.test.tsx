import { describe, expect, it, afterEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { i18n } from '@/libs/i18n';
import { LanguageSwitcher } from './LanguageSwitcher';

describe('LanguageSwitcher', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('renders language combobox', () => {
    renderWithTheme(<LanguageSwitcher />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('still renders combobox after unsupported language', async () => {
    await i18n.changeLanguage('de');
    renderWithTheme(<LanguageSwitcher />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });
});
