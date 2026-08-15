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
import {
  rtkErrorMessage,
  useCreatePriceAlertMutation,
  useDeletePriceAlertMutation,
  useListPriceAlertsQuery,
  type MarketExchange,
} from '@/libs/api';
import { PageStack, Section } from './AlertsPage.styles';

const EXCHANGES = new Set(['binance', 'coinbase', 'bybit', 'nasdaq', 'bist']);

function parsePrefillExchange(raw: string | null): MarketExchange {
  const v = (raw ?? '').trim().toLowerCase();
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

  const defaultExchange = useMemo(
    () => parsePrefillExchange(searchParams.get('exchange')),
    [searchParams],
  );
  const defaultSymbol = useMemo(
    () => (searchParams.get('symbol') ?? '').trim().toUpperCase(),
    [searchParams],
  );

  const items = list.data?.alerts ?? [];

  return (
    <PageStack>
      <PageHeader title={t('alerts:title')} />

      <Section>
        <CreateAlertForm
          key={`${defaultExchange}:${defaultSymbol}`}
          defaultExchange={defaultExchange}
          defaultSymbol={defaultSymbol}
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
