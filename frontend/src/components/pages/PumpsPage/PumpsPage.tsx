import { useMemo, useState } from 'react';
import { Alert, Button, Empty, Select, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  rtkErrorMessage,
  useScanPumpEventsQuery,
  type MarketExchange,
} from '@/libs/api';
import { defaultQuoteForExchange, formatSymbolDisplay } from '@/libs/utils';
import {
  pumpScanHitsToRows,
  type PumpScanHitWire,
  type PumpScanRow,
} from './PumpsPage.helpers';
import { Field, PageIntro, PageStack, Toolbar } from './PumpsPage.styles';

export function PumpsPage() {
  const { t } = useTranslation(['pumps', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<MarketExchange>('binance');
  const [quote, setQuote] = useState(defaultQuoteForExchange('binance'));
  const [interval, setInterval] = useState('15m');
  const [scanKey, setScanKey] = useState(0);

  const scan = useScanPumpEventsQuery(
    {
      exchange,
      quote,
      interval,
      symbolLimit: 20,
      // Slightly lower than API default 8 so scans return more usable hits in calm markets.
      minReturnPct: 5,
    },
    { skip: scanKey === 0 },
  );

  const rows = useMemo(() => {
    const hits = (scan.data as { hits?: PumpScanHitWire[] } | undefined)?.hits;
    return pumpScanHitsToRows(hits);
  }, [scan.data]);

  const hitCount =
    (scan.data as { hitCount?: number } | undefined)?.hitCount ?? rows.length;

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
          <Text variant="caption" color="secondary">
            {t('pumps:exchange')}
          </Text>
          <Select
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
          <Text variant="caption" color="secondary">
            {t('pumps:quote')}
          </Text>
          <Select
            value={quote}
            style={{ minWidth: 100 }}
            options={['USDT', 'USD', 'USDC', 'EUR'].map((q) => ({ value: q, label: q }))}
            onChange={setQuote}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('pumps:interval')}
          </Text>
          <Select
            value={interval}
            style={{ minWidth: 100 }}
            options={['5m', '15m', '1h', '4h'].map((iv) => ({ value: iv, label: iv }))}
            onChange={setInterval}
          />
        </Field>
        <Button
          type="primary"
          loading={scan.isFetching}
          onClick={() => setScanKey((k) => k + 1)}
        >
          {t('pumps:scan')}
        </Button>
      </Toolbar>

      {scanKey > 0 && scan.isSuccess ? (
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
        dataSource={scanKey === 0 ? [] : rows}
        columns={columns}
        pagination={{ pageSize: 20 }}
        onRow={(record) => ({
          onClick: () => {
            navigate(
              `/markets/${encodeURIComponent(record.exchange)}/${encodeURIComponent(record.symbol)}`,
            );
          },
          style: { cursor: 'pointer' },
        })}
        locale={{
          emptyText: (
            <Empty
              description={
                scanKey === 0 ? t('pumps:emptyHint') : t('pumps:emptyTitle')
              }
            />
          ),
        }}
      />
    </PageStack>
  );
}
