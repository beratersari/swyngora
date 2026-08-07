import { useEffect, useMemo, useState } from 'react';
import { AutoComplete } from 'antd';
import { useTranslation } from 'react-i18next';
import { useLazyListSpotMarketsQuery, type MarketExchange } from '@/libs/api';
import { defaultQuoteForExchange, formatSymbolDisplay } from '@/libs/utils';
import { useDebouncedValue } from '@/libs/hooks';
import type { SymbolSuggestProps } from './SymbolSuggest.types';

/**
 * Typeahead for spot symbols on a venue (markets list search).
 * Presentational input — parents own exchange + selected symbol state.
 */
export function SymbolSuggest({
  exchange,
  value,
  onChange,
  onPick,
  placeholder = 'BTCUSDT',
  disabled,
  'aria-label': ariaLabel,
  style,
}: SymbolSuggestProps) {
  const { t } = useTranslation(['common', 'markets']);
  const [query, setQuery] = useState(value);
  const debounced = useDebouncedValue(query, 250);
  const [fetchSpot, spotState] = useLazyListSpotMarketsQuery();

  useEffect(() => {
    setQuery(value);
  }, [value]);

  useEffect(() => {
    const q = debounced.trim();
    if (q.length < 1) return;
    void fetchSpot({
      exchange: exchange as MarketExchange,
      q,
      quote: defaultQuoteForExchange(exchange),
      sort: 'quoteVolume',
      order: 'desc',
      limit: 12,
      offset: 0,
      status: 'TRADING',
    });
  }, [debounced, exchange, fetchSpot]);

  const options = useMemo(() => {
    const items = spotState.data?.items ?? [];
    return items
      .filter((it) => it.symbol)
      .map((it) => ({
        value: it.symbol!,
        label: formatSymbolDisplay(it.symbol),
      }));
  }, [spotState.data?.items]);

  return (
    <AutoComplete
      value={query}
      options={options}
      disabled={disabled}
      style={{ minWidth: 160, ...style }}
      placeholder={placeholder}
      aria-label={ariaLabel ?? t('markets:toolbar.search', { defaultValue: 'Symbol' })}
      onSearch={(text) => {
        const next = text.toUpperCase();
        setQuery(next);
        onChange(next);
      }}
      onSelect={(v) => {
        const next = String(v).toUpperCase();
        setQuery(next);
        onChange(next);
        onPick?.(next);
      }}
      onChange={(text) => {
        const next = String(text).toUpperCase();
        setQuery(next);
        onChange(next);
      }}
      filterOption={false}
      notFoundContent={
        spotState.isFetching
          ? t('common:status.loading')
          : debounced.trim()
            ? t('markets:table.emptyTitle', { defaultValue: 'No matches' })
            : null
      }
    />
  );
}
