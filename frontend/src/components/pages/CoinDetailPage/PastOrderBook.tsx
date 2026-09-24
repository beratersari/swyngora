import { useState } from 'react';
import { Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { useGetOrderBookHistoryQuery } from '@/libs/api';
import { formatCompactUsd } from '@/libs/utils';

type PastOrderBookProps = {
  symbol: string;
  exchange: string;
};

export function PastOrderBook({ symbol, exchange }: PastOrderBookProps) {
  const { t } = useTranslation('detail');
  const list = useGetOrderBookHistoryQuery({ symbol, exchange, limit: 30 });
  const snaps = list.data?.snapshots ?? [];
  const [at, setAt] = useState<string>();
  const chosen = at || snaps[0]?.sampledAt;
  const one = useGetOrderBookHistoryQuery(
    { symbol, exchange, at: chosen },
    { skip: !chosen },
  );
  const flashes = one.data?.snapshot?.flashes ?? [];

  return (
    <section data-testid="past-order-book">
      <Text variant="h4" color="primary">
        {t('pastBook.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('pastBook.hint')}
      </Text>
      <Select
        aria-label={t('pastBook.when')}
        style={{ minWidth: 220, marginTop: 8 }}
        value={chosen}
        loading={list.isFetching}
        options={snaps.map((row) => ({
          value: row.sampledAt,
          label: row.sampledAt ? new Date(row.sampledAt).toLocaleString() : '—',
        }))}
        onChange={setAt}
        placeholder={t('pastBook.empty')}
      />
      {chosen && flashes.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('pastBook.none')}
        </Text>
      ) : null}
      {flashes.map((row) => (
        <Text key={`${row.side}-${row.price}-${row.seenFrom}`} variant="body" color="primary">
          {t('pastBook.flash', {
            side: row.side === 'ask' ? t('orderBook.askHeavy') : t('orderBook.bidHeavy'),
            price: row.price,
            notional: formatCompactUsd(row.notional),
            seconds: Math.max(1, Math.round(row.durationSec ?? 0)),
          })}
        </Text>
      ))}
    </section>
  );
}
