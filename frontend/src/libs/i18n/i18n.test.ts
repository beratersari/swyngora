import { beforeEach, describe, expect, it } from 'vitest';
import { i18n } from './i18n';

describe('i18n', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('loads english markets title', () => {
    expect(i18n.t('markets:title')).toBe('Markets');
  });

  it('switches to turkish', async () => {
    await i18n.changeLanguage('tr');
    expect(i18n.t('markets:title')).toBe('Piyasalar');
    expect(i18n.t('common:actions.retry')).toBe('Yeniden dene');
  });

  it('interpolates range params', () => {
    expect(
      i18n.t('markets:results.range', { from: '1', to: '50', total: '120' }),
    ).toBe('Showing 1–50 of 120');
  });
});
