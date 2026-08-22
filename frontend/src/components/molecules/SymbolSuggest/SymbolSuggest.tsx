import { useEffect, useMemo, useState } from 'react';
import { AutoComplete } from 'antd';
import { useTranslation } from 'react-i18next';
import { useLazyListSpotMarketsQuery, type MarketExchange } from '@/libs/api';
import { formatSymbolDisplay, rtkCurrent } from '@/libs/utils';
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
  className,
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
      sort: 'quoteVolume',
      order: 'desc',
      limit: 12,
      offset: 0,
      status: 'TRADING',
    });
  }, [debounced, exchange, fetchSpot]);

  const options = useMemo(() => {
    const args = spotState.originalArgs;
    if (!args || args.exchange !== exchange) return [];
    const items = rtkCurrent(spotState)?.items ?? [];
    return items
      .filter((it) => it.symbol)
      .map((it) => ({
        value: it.symbol!,
        label: formatSymbolDisplay(it.symbol),
      }));
  }, [spotState, exchange]);

  return (
    <AutoComplete
      className={className}
      value={query}
      options={options}
      disabled={disabled}
      style={{ minWidth: 0, width: '100%', maxWidth: '100%', ...style }}
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
