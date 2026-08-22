import { Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { useDisplayCurrency } from '@/libs/hooks';
import { DISPLAY_CURRENCIES } from '@/libs/utils';
import type { CurrencySwitcherProps } from './CurrencySwitcher.types';

/**
 * Display-currency switcher. Converts BIST TRY / Nasdaq USD / crypto USDT
 * using GET /api/v1/market/fx. Persists in localStorage.
 */
export function CurrencySwitcher({ className, size = 'small' }: CurrencySwitcherProps) {
  const { t } = useTranslation('common');
  const { currency, setCurrency, stale } = useDisplayCurrency();

  return (
    <Select
      className={className}
      size={size}
      value={currency}
      aria-label={stale ? `${t('currency.label')} — ${t('currency.stale')}` : t('currency.label')}
      status={stale ? 'warning' : undefined}
      title={stale ? t('currency.stale') : undefined}
      style={{ minWidth: 118 }}
      options={DISPLAY_CURRENCIES.map((code) => ({
        value: code,
        label: t(`currency.${code}`),
      }))}
      onChange={(next) => setCurrency(next)}
    />
  );
}
