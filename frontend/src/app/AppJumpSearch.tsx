import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';

/** Header jump-to-market search. RTK stays in SymbolSuggest (existing pattern). */
export function AppJumpSearch() {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const [symbol, setSymbol] = useState('');

  return (
    <SymbolSuggest
      exchange="binance"
      value={symbol}
      placeholder={t('search.placeholder')}
      aria-label={t('search.aria')}
      style={{ width: 200 }}
      onChange={setSymbol}
      onPick={(next) => {
        const raw = next.trim().toUpperCase();
        if (!raw) return;
        navigate(`/markets/binance/${encodeURIComponent(raw)}`);
      }}
    />
  );
}
