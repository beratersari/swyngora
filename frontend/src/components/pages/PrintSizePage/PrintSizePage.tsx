import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { PrintSizeTable } from '@/components/organisms/PrintSizeTable';
import { rtkErrorMessage, useGetMarketPrintSizeOverviewQuery } from '@/libs/api';
import {
  PRINT_SIZE_EXCHANGES,
  PRINT_SIZE_LIMIT,
  PRINT_SIZE_WINDOWS,
  type PrintSizeExchange,
  type PrintSizeWindowId,
} from './constants';
import { coinsToRows, windowById } from './helpers';
import { Field, PageStack, Toolbar, WindowCard, WindowGrid } from './PrintSizePage.styles';

export function PrintSizePage() {
  const { t } = useTranslation(['printsize', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<PrintSizeExchange>('all');
  const [windowId, setWindowId] = useState<PrintSizeWindowId>('15m');
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useGetMarketPrintSizeOverviewQuery(
    { exchange, quote: 'USDT', limit: PRINT_SIZE_LIMIT },
    { skip: !hasLoaded },
  );

  const readLabel = (read: string) => {
    if (read === 'few_large') return t('printsize:fewLarge');
    if (read === 'many_small') return t('printsize:manySmall');
    if (read === 'larger') return t('printsize:larger');
    if (read === 'smaller') return t('printsize:smaller');
    if (read === 'stable') return t('printsize:stable');
    return '—';
  };

  const rows = useMemo(
    () => coinsToRows(query.data?.coins, windowId, readLabel),
    [query.data?.coins, windowId, t],
  );
  const active = windowById(query.data?.windows, windowId);

  return (
    <PageStack>
      <PageHeader title={t('printsize:title')} subtitle={t('printsize:subtitle')} />

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="printsize-exchange-label">
            {t('printsize:exchange')}
          </Text>
          <Select
            aria-labelledby="printsize-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={PRINT_SIZE_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('printsize:allVenues') : t(`common:exchanges.${value}`),
            }))}
            onChange={(value) => {
              setExchange(value);
              setHasLoaded(false);
            }}
          />
        </Field>
        <Button
          type="primary"
          loading={query.isFetching}
          onClick={() => {
            if (!hasLoaded) {
              setHasLoaded(true);
              return;
            }
            void query.refetch();
          }}
        >
          {t('printsize:load')}
        </Button>
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('printsize:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('printsize:resource') })}
        />
      ) : null}

      {hasLoaded && query.isSuccess ? (
        <WindowGrid>
          {PRINT_SIZE_WINDOWS.map((id) => {
            const win = windowById(query.data?.windows, id);
            return (
              <WindowCard
                key={id}
                type="button"
                $active={id === windowId}
                onClick={() => setWindowId(id)}
              >
                <Text variant="caption" color="secondary">
                  {id}
                </Text>
                <Text variant="label" color="primary">
                  {t('printsize:buyLine', {
                    avg: win?.buy?.avg || '—',
                    prior: win?.buy?.priorAvg || '—',
                  })}
                </Text>
                <Text variant="caption" color="secondary">
                  {t('printsize:sellLine', {
                    avg: win?.sell?.avg || '—',
                    prior: win?.sell?.priorAvg || '—',
                  })}
                </Text>
                <Text variant="caption" color="secondary">
                  {t('printsize:coinSplit', {
                    larger: win?.largerCoins ?? 0,
                    smaller: win?.smallerCoins ?? 0,
                    few: win?.fewLargeCoins ?? 0,
                    many: win?.manySmallCoins ?? 0,
                  })}
                </Text>
              </WindowCard>
            );
          })}
        </WindowGrid>
      ) : null}

      {active?.summary ? (
        <Text variant="caption" color="secondary">
          {active.summary}
        </Text>
      ) : null}

      <PrintSizeTable
        rows={rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded}
        emptyHint={t('printsize:emptyHint')}
        emptyTitle={t('printsize:emptyTitle')}
        columns={{
          symbol: t('printsize:symbol'),
          buyAvg: t('printsize:buyAvg'),
          buyPrior: t('printsize:buyPrior'),
          sellAvg: t('printsize:sellAvg'),
          sellPrior: t('printsize:sellPrior'),
          read: t('printsize:read'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(
            `/markets/${encodeURIComponent(ex === 'all' ? 'binance' : ex)}/${encodeURIComponent(symbol)}?tab=tape`,
          )
        }
      />
    </PageStack>
  );
}
