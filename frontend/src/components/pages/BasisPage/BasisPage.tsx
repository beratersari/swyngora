import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { BasisCurveHistory } from '@/components/organisms/BasisCurveHistory';
import { BasisPanel, BASIS_POLL_MS, DEFAULT_BASIS_SYMBOL } from '@/components/organisms/BasisPanel';
import { rtkErrorMessage, useGetMarketBasisQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { rtkCurrent } from '@/libs/utils';
import { Field, PageStack, Toolbar } from './BasisPage.styles';
import { Text } from '@/components/atoms/Text';

export function BasisPage() {
  const { t } = useTranslation(['basis', 'common']);
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const symbol = (searchParams.get('symbol') || DEFAULT_BASIS_SYMBOL).toUpperCase();

  const query = useGetMarketBasisQuery(
    { symbol, exchange: 'all' },
    { pollingInterval: visible ? BASIS_POLL_MS : 0, refetchOnFocus: true },
  );
  const report = rtkCurrent(query);

  return (
    <PageStack>
      <PageHeader title={t('basis:title')} subtitle={t('basis:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary">
            {t('basis:symbol')}
          </Text>
          <SymbolSuggest
            exchange="binance"
            value={symbol}
            onChange={(next) => {
              setSearchParams(
                (prev) => {
                  const p = new URLSearchParams(prev);
                  p.set('symbol', next);
                  return p;
                },
                { replace: true },
              );
            }}
          />
        </Field>
      </Toolbar>
      <BasisPanel
        report={report}
        isLoading={query.isLoading}
        errorMessage={query.isError ? rtkErrorMessage(query.error) : null}
      />
      {(report?.venues ?? []).map((v) => (
        <BasisCurveHistory
          key={`curve-hist-${v.exchange ?? v.symbol}`}
          curve={v.curve}
          venueLabel={v.exchange ? v.exchange.charAt(0).toUpperCase() + v.exchange.slice(1) : undefined}
        />
      ))}
    </PageStack>
  );
}
