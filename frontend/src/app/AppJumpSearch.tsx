import { useMemo, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { parseExchangeParamOrDefault } from '@/libs/utils';

/** Header jump-to-market search. Venue follows the current markets route. */
export function AppJumpSearch() {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const [searchParams] = useSearchParams();
  const [symbol, setSymbol] = useState('');

  const exchange = useMemo(() => {
    const match = pathname.match(/^\/markets\/([^/]+)/);
    if (match?.[1]) {
      return parseExchangeParamOrDefault(match[1]);
    }
    if (pathname === '/markets' || pathname === '/heatmap') {
      return parseExchangeParamOrDefault(searchParams.get('exchange') ?? undefined);
    }
    return parseExchangeParamOrDefault(undefined);
  }, [pathname, searchParams]);

  return (
    <SymbolSuggest
      className="desk-jump-search"
      exchange={exchange}
      value={symbol}
      placeholder={t('search.placeholder')}
      aria-label={t('search.aria')}
      style={{ width: 220, maxWidth: '100%' }}
      onChange={setSymbol}
      onPick={(next) => {
        const raw = next.trim().toUpperCase();
        if (!raw) return;
        navigate(`/markets/${encodeURIComponent(exchange)}/${encodeURIComponent(raw)}`);
      }}
    />
  );
}
