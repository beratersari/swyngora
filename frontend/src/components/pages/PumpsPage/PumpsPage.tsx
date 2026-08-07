import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Empty, Select, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  rtkErrorMessage,
  useListIntervalsQuery,
  useScanPumpEventsQuery,
  type MarketExchange,
} from '@/libs/api';
import { defaultQuoteForExchange, formatSymbolDisplay } from '@/libs/utils';
import { pumpScanHitsToRows, type PumpScanRow } from './PumpsPage.helpers';
import { Field, PageIntro, PageStack, Toolbar } from './PumpsPage.styles';

/** Prefer 15m when supported; otherwise first venue interval. */
function pickDefaultInterval(intervals: string[] | undefined, current: string): string {
  if (!intervals?.length) return current;
  if (intervals.includes(current)) return current;
  if (intervals.includes('15m')) return '15m';
  if (intervals.includes('1h')) return '1h';
  return intervals[0]!;
}

export function PumpsPage() {
  const { t } = useTranslation(['pumps', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<MarketExchange>('binance');
  const [quote, setQuote] = useState(defaultQuoteForExchange('binance'));
  const [interval, setInterval] = useState('15m');
  /** False until the user runs the first scan (skip query until then). */
  const [hasScanned, setHasScanned] = useState(false);

  const intervalsQuery = useListIntervalsQuery({ exchange });
  const intervalOptions = intervalsQuery.data?.intervals ?? [];

  useEffect(() => {
    setInterval((prev) => pickDefaultInterval(intervalOptions, prev));
  }, [exchange, intervalOptions]);

  const scan = useScanPumpEventsQuery(
    {
      exchange,
      quote,
      interval,
      symbolLimit: 20,
      // Slightly lower than API default 8 so scans return more usable hits in calm markets.
      minReturnPct: 5,
    },
    { skip: !hasScanned },
  );

  const rows = useMemo(() => pumpScanHitsToRows(scan.data?.hits), [scan.data]);

  const hitCount = scan.data?.hitCount ?? rows.length;

  const columns: ColumnsType<PumpScanRow> = [
    {
      title: t('pumps:symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      render: (s: string) => (
        <Text variant="label" mono color="primary">
          {formatSymbolDisplay(s)}
        </Text>
      ),
    },
    {
      title: t('pumps:returnPct'),
      dataIndex: 'returnPct',
      key: 'returnPct',
      align: 'right',
      render: (v: number | null) => (
        <Text variant="numeric">
          {v != null && Number.isFinite(v) ? `${v.toFixed(2)}%` : '—'}
        </Text>
      ),
    },
    {
      title: t('pumps:volumeRatio'),
      dataIndex: 'volumeRatio',
      key: 'volumeRatio',
      align: 'right',
      render: (v: number | null) => (
        <Text variant="numeric">
          {v != null && Number.isFinite(v) ? v.toFixed(2) : '—'}
        </Text>
      ),
    },
    {
      title: t('pumps:time'),
      dataIndex: 'openTime',
      key: 'openTime',
      render: (v: string | null) => (
        <Text variant="caption" color="secondary">
          {v ? new Date(v).toLocaleString() : '—'}
        </Text>
      ),
    },
    {
      title: t('pumps:events', { defaultValue: 'Events' }),
      dataIndex: 'eventCount',
      key: 'eventCount',
      align: 'right',
      render: (n: number) => <Text variant="numeric">{n}</Text>,
    },
  ];

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="primary">
          {t('pumps:title')}
        </Text>
        <Text variant="body" color="secondary">
          {t('pumps:subtitle')}
        </Text>
      </PageIntro>

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="pumps-exchange-label">
            {t('pumps:exchange')}
          </Text>
          <Select
            aria-labelledby="pumps-exchange-label"
            value={exchange}
            style={{ minWidth: 120 }}
            options={[
              { value: 'binance', label: 'binance' },
              { value: 'coinbase', label: 'coinbase' },
              { value: 'bybit', label: 'bybit' },
            ]}
            onChange={(v) => {
              setExchange(v);
              setQuote(defaultQuoteForExchange(v));
            }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="pumps-quote-label">
            {t('pumps:quote')}
          </Text>
          <Select
            aria-labelledby="pumps-quote-label"
            value={quote}
            style={{ minWidth: 100 }}
            options={['USDT', 'USD', 'USDC', 'EUR'].map((q) => ({ value: q, label: q }))}
            onChange={setQuote}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="pumps-interval-label">
            {t('pumps:interval')}
          </Text>
          <Select
            aria-labelledby="pumps-interval-label"
            value={interval}
            style={{ minWidth: 100 }}
            loading={intervalsQuery.isLoading}
            options={(intervalOptions.length
              ? intervalOptions
              : ['5m', '15m', '1h']
            ).map((iv) => ({ value: iv, label: iv }))}
            onChange={setInterval}
          />
        </Field>
        <Button
          type="primary"
          loading={scan.isFetching}
          onClick={() => {
            if (!hasScanned) {
              setHasScanned(true);
              return;
            }
            // Same filters: force a new network request (RTK cache key is args only).
            void scan.refetch();
          }}
        >
          {t('pumps:scan')}
        </Button>
      </Toolbar>

      {hasScanned && scan.isSuccess ? (
        <Text variant="caption" color="secondary">
          {t('pumps:hits', { count: hitCount })} · {t('pumps:note')}
        </Text>
      ) : null}

      {scan.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('pumps:loadFailed')}
          description={rtkErrorMessage(scan.error, { resource: t('pumps:resource') })}
        />
      ) : null}

      <Table
        rowKey={(r) => `${r.exchange}:${r.symbol}:${r.openTime ?? r.returnPct}`}
        loading={scan.isFetching}
        dataSource={hasScanned ? rows : []}
        columns={columns}
        pagination={{ pageSize: 20 }}
        onRow={(record) => {
          const open = () => {
            navigate(
              `/markets/${encodeURIComponent(record.exchange)}/${encodeURIComponent(record.symbol)}`,
            );
          };
          return {
            onClick: open,
            onKeyDown: (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                open();
              }
            },
            tabIndex: 0,
            role: 'link',
            'aria-label': t('pumps:openDetail', {
              defaultValue: 'Open {{symbol}}',
              symbol: formatSymbolDisplay(record.symbol),
            }),
            style: { cursor: 'pointer' },
          };
        }}
        locale={{
          emptyText: (
            <Empty
              description={
                !hasScanned ? t('pumps:emptyHint') : t('pumps:emptyTitle')
              }
            />
          ),
        }}
      />
    </PageStack>
  );
}
