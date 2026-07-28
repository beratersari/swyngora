import { describe, expect, it } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { initI18n, setAppLocale } from '@/libs/i18n';
import { LanguageSwitcher } from './LanguageSwitcher';

describe('LanguageSwitcher', () => {
  it('renders locales and switches language', async () => {
    initI18n();
    await setAppLocale('en');
    render(<LanguageSwitcher />);
    expect(screen.getByText('Language')).toBeTruthy();
    expect(screen.getByText('English')).toBeTruthy();
    fireEvent.click(screen.getByText('Türkçe'));
    // After switch, common label becomes Turkish
    expect(await screen.findByText('Dil')).toBeTruthy();
    await setAppLocale('en');
  });
});
