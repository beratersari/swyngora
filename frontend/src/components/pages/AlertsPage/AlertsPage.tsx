import { useMemo, useState } from 'react';
import { Alert, Button, message } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { AlertsTable } from '@/components/organisms/AlertsTable';
import {
  CreateAlertForm,
  type CreatePriceAlertValues,
} from '@/components/organisms/CreateAlertForm/CreateAlertForm';
import type { AlertKind } from '@/components/organisms/CreateAlertForm/CreateAlertForm.types';
import {
  rtkErrorMessage,
  useCreatePriceAlertMutation,
  useDeletePriceAlertMutation,
  useListPriceAlertsQuery,
  type MarketExchange,
} from '@/libs/api';
import { PageStack, Section } from './AlertsPage.styles';

const EXCHANGES = new Set(['binance', 'coinbase', 'bybit', 'nasdaq', 'bist']);
const RADAR_KINDS = new Set<AlertKind>([
  'price',
  'liquidation_feed',
  'liquidation_cascade',
  'liquidation_notional',
  'liquidation_level',
  'setup_radar',
  'funding',
  'open_interest',
  'taker',
  'fear_greed',
]);
const RADAR_EXCHANGES = new Set(['binance', 'bybit', 'okx', 'all']);

function parsePrefillKind(raw: string | null): AlertKind {
  const v = (raw ?? '').trim().toLowerCase();
  return RADAR_KINDS.has(v as AlertKind) ? (v as AlertKind) : 'price';
}

function parsePrefillExchange(raw: string | null, kind: AlertKind): MarketExchange | string {
  const v = (raw ?? '').trim().toLowerCase();
  if (kind === 'setup_radar' || kind.startsWith('liquidation') || kind === 'funding' || kind === 'open_interest' || kind === 'taker' || kind === 'fear_greed') {
    return RADAR_EXCHANGES.has(v) ? v : 'all';
  }
  return EXCHANGES.has(v) ? (v as MarketExchange) : 'binance';
}

/**
 * Alerts list + create against backend price-alert API (server-side evaluation).
 */
export function AlertsPage() {
  const { t } = useTranslation(['alerts', 'common']);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const list = useListPriceAlertsQuery();
  const [create, createState] = useCreatePriceAlertMutation();
  const [del, delState] = useDeletePriceAlertMutation();
  const [actionError, setActionError] = useState<string | null>(null);

  const defaultKind = useMemo(() => parsePrefillKind(searchParams.get('kind')), [searchParams]);
  const defaultExchange = useMemo(
    () => parsePrefillExchange(searchParams.get('exchange'), defaultKind),
    [searchParams, defaultKind],
  );
  const defaultSymbol = useMemo(() => {
    const raw = (searchParams.get('symbol') ?? '').trim().toUpperCase();
    if (defaultKind === 'setup_radar' && (raw === '' || raw === 'ALL')) return '';
    return raw;
  }, [searchParams, defaultKind]);

  const items = list.data?.alerts ?? [];

  return (
    <PageStack>
      <PageHeader title={t('alerts:title')} />

      <Section>
        <CreateAlertForm
          key={`${defaultKind}:${defaultExchange}:${defaultSymbol}`}
          defaultExchange={defaultExchange}
          defaultSymbol={defaultSymbol}
          defaultKind={defaultKind}
          isSubmitting={createState.isLoading}
          submitError={createState.isError ? createState.error : undefined}
          onSubmit={async (values: CreatePriceAlertValues) => {
            setActionError(null);
            await create(values).unwrap();
            void message.success(t('alerts:createSuccess', { defaultValue: 'Alert created' }));
            void list.refetch();
          }}
        />
      </Section>

      {list.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('alerts:loadFailed')}
          description={rtkErrorMessage(list.error, { resource: t('alerts:resource') })}
          action={
            <Button size="small" onClick={() => void list.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      {actionError ? (
        <Alert type="error" showIcon message={actionError} closable onClose={() => setActionError(null)} />
      ) : null}

      <Section>
        <Text variant="h4" color="primary">
          {t('alerts:listTitle')}
        </Text>
        <AlertsTable
          items={items}
          loading={list.isLoading}
          deleteLoading={delState.isLoading}
          onDelete={(id) => {
            setActionError(null);
            void del({ id })
              .unwrap()
              .catch((err: unknown) => {
                setActionError(rtkErrorMessage(err, { resource: t('alerts:resource') }));
              });
          }}
          onOpen={(exchange, symbol) => {
            navigate(
              `/markets/${encodeURIComponent(exchange)}/${encodeURIComponent(symbol)}`,
            );
          }}
        />
      </Section>
    </PageStack>
  );
}
